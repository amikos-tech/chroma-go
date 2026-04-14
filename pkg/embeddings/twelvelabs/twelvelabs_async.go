package twelvelabs

import (
	"context"
	"time"

	"github.com/pkg/errors"

	chttp "github.com/amikos-tech/chroma-go/pkg/commons/http"
	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

// contentToAsyncRequest builds the tasks-endpoint body for audio/video content.
// See RESEARCH F-02 — the async endpoint uses embedding_option as []string
// and does NOT accept "fused" (only "audio" and "transcription").
func contentToAsyncRequest(content embeddings.Content, model string, audioOpt string) (*AsyncEmbedV2Request, error) {
	if len(content.Parts) != 1 {
		return nil, errors.Errorf("Twelve Labs requires exactly one part per Content item, got %d", len(content.Parts))
	}
	part := content.Parts[0]
	req := &AsyncEmbedV2Request{ModelName: model}
	switch part.Modality {
	case embeddings.ModalityAudio:
		// RESEARCH F-02 / A5: "fused" is valid on the sync embedding_option
		// string but is NOT a valid async embedding_option list value. Reject
		// deterministically rather than silently dropping or mapping.
		if audioOpt == "fused" {
			return nil, errors.New("Twelve Labs async path does not support audio embedding option \"fused\"; async endpoint only accepts \"audio\" and \"transcription\" (see RESEARCH F-02). Disable WithAsyncPolling for fused-audio calls.")
		}
		ms, err := buildMediaSource(part.Source)
		if err != nil {
			return nil, errors.Wrap(err, "audio source")
		}
		req.InputType = "audio"
		audio := &AsyncAudioInput{MediaSource: ms}
		if audioOpt != "" {
			audio.EmbeddingOption = []string{audioOpt} // wrap single-string sync option into list per F-02
		}
		req.Audio = audio
	case embeddings.ModalityVideo:
		ms, err := buildMediaSource(part.Source)
		if err != nil {
			return nil, errors.Wrap(err, "video source")
		}
		req.InputType = "video"
		req.Video = &AsyncVideoInput{MediaSource: ms}
	default:
		return nil, errors.Errorf("async path only handles audio/video; got modality %q", part.Modality)
	}
	return req, nil
}

// pollTask loops GET /tasks/{id} until ready, failed, ctx-cancel, or maxWait-expiry.
// D-09/D-10/D-11/D-14/D-16/D-17/D-20 all land here.
//
// Per-HTTP-call deadline: every doTaskGet is wrapped in a derived context
// bounded by min(parent ctx deadline, SDK maxWait deadline). This ensures
// a blocked HTTP request cannot outlive maxWait (D-09 hard bound). When the
// derived deadline fires, we inspect which source expired first and translate
// back into the appropriate distinct error message (D-20).
func (e *TwelveLabsEmbeddingFunction) pollTask(ctx context.Context, taskID string, maxWait time.Duration) (*TaskResponse, error) {
	sdkMaxWaitDeadline := time.Now().Add(maxWait)
	interval := e.apiClient.asyncPollInitial
	for {
		// Derive per-call deadline = min(parent ctx deadline, sdkMaxWaitDeadline).
		callDeadline := sdkMaxWaitDeadline
		if parentDL, ok := ctx.Deadline(); ok && parentDL.Before(callDeadline) {
			callDeadline = parentDL
		}
		callCtx, cancel := context.WithDeadline(ctx, callDeadline)
		resp, err := e.doTaskGet(callCtx, taskID)
		cancel()
		if err != nil {
			// If the call was terminated because our derived deadline fired,
			// figure out which source was responsible and return the distinct
			// error (D-20). errors.Is works through pkg/errors wrapping.
			if errors.Is(err, context.DeadlineExceeded) {
				if time.Now().After(sdkMaxWaitDeadline) {
					return nil, errors.Errorf("Twelve Labs task [%s] async polling maxWait %s exceeded", taskID, maxWait)
				}
				// Parent ctx deadline fired first.
				return nil, errors.Wrap(ctx.Err(), "Twelve Labs async polling deadline exceeded")
			}
			if errors.Is(err, context.Canceled) {
				return nil, errors.Wrap(ctx.Err(), "Twelve Labs async polling canceled")
			}
			return nil, err
		}
		switch resp.Status {
		case "ready":
			return resp, nil
		case "failed":
			// Use the raw server body captured by doTaskGet (Plan 01 FailureDetail)
			// so the sanitized reason reflects the authentic server payload,
			// not a re-marshaled subset of known fields.
			return nil, errors.Errorf("Twelve Labs task [%s] terminal status=failed: %s", taskID, chttp.SanitizeErrorBody(resp.FailureDetail))
		case "processing":
			// fall through to sleep
		default:
			return nil, errors.Errorf("Twelve Labs task [%s] unexpected status %q", taskID, resp.Status)
		}

		remaining := time.Until(sdkMaxWaitDeadline)
		if remaining <= 0 {
			return nil, errors.Errorf("Twelve Labs task [%s] async polling maxWait %s exceeded", taskID, maxWait)
		}
		wait := interval
		if wait > remaining {
			wait = remaining
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			// Distinct wording for cancel vs deadline (D-20). errors.Is still
			// unwraps to the stdlib sentinel for caller introspection.
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, errors.Wrap(ctx.Err(), "Twelve Labs async polling deadline exceeded")
			}
			return nil, errors.Wrap(ctx.Err(), "Twelve Labs async polling canceled")
		case <-timer.C:
		}

		interval = nextBackoff(interval, e.apiClient.asyncPollMultiplier, e.apiClient.asyncPollCap)
	}
}

func nextBackoff(cur time.Duration, mul float64, backoffCap time.Duration) time.Duration {
	next := time.Duration(float64(cur) * mul)
	if next > backoffCap {
		next = backoffCap
	}
	return next
}

// createTaskAndPoll builds the async request, creates the task, and polls to completion.
func (e *TwelveLabsEmbeddingFunction) createTaskAndPoll(ctx context.Context, content embeddings.Content) (embeddings.Embedding, error) {
	req, err := contentToAsyncRequest(content, e.resolveModel(ctx), e.apiClient.AudioEmbeddingOption)
	if err != nil {
		return nil, err
	}
	created, err := e.doTaskPost(ctx, *req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Twelve Labs async task")
	}
	if created.ID == "" {
		return nil, errors.New("Twelve Labs async task create returned empty _id")
	}
	// Rare early-ready path (server finished before response round-trip returned).
	if created.Status == "ready" && len(created.Data) > 0 {
		return buildEmbeddingFromData(created.Data)
	}
	final, err := e.pollTask(ctx, created.ID, e.apiClient.asyncMaxWait)
	if err != nil {
		return nil, err
	}
	return buildEmbeddingFromData(final.Data)
}

// buildEmbeddingFromData mirrors embeddingFromResponse but operates on a raw data slice.
func buildEmbeddingFromData(data []EmbedV2DataItem) (embeddings.Embedding, error) {
	if len(data) == 0 {
		return nil, errors.New("no embedding returned from Twelve Labs task")
	}
	if len(data) > 1 {
		return nil, errors.Errorf("expected 1 embedding from Twelve Labs task, got %d", len(data))
	}
	if len(data[0].Embedding) == 0 {
		return nil, errors.New("empty embedding vector returned from Twelve Labs task")
	}
	return embeddings.NewEmbeddingFromFloat32(float64sToFloat32s(data[0].Embedding)), nil
}
