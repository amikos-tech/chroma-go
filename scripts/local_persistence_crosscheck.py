#!/usr/bin/env python3
"""
Cross-language local persistence harness for chroma-go-local compatibility checks.

This script is intentionally small and deterministic. It operates directly on a
PersistentClient path so Go tests can validate roundtrips against the same local
on-disk Chroma data.
"""

import argparse
import json
import sys

import chromadb


GO_QUERY_EMBEDDING = [0.95, 0.05, 0.0]
PY_QUERY_EMBEDDING = [0.05, 0.95, 0.0]


def _extract_top_id(query_result) -> str:
    ids = query_result.get("ids") or []
    if not ids:
        return ""
    first_group = ids[0] or []
    if not first_group:
        return ""
    return first_group[0]


def verify_go_collection(persist_path: str, collection_name: str) -> dict:
    client = chromadb.PersistentClient(path=persist_path)
    collection = client.get_collection(collection_name)

    count = collection.count()
    query = collection.query(
        query_embeddings=[GO_QUERY_EMBEDDING],
        n_results=1,
        include=["documents", "distances"],
    )
    top_id = _extract_top_id(query)

    updated_id = "go-2"
    updated_document = "go document 2 updated by python"
    collection.update(
        ids=[updated_id],
        documents=[updated_document],
        embeddings=[[0.0, 1.0, 0.0]],
    )

    get_result = collection.get(ids=[updated_id], include=["documents"])
    persisted_document = ""
    docs = get_result.get("documents") or []
    if docs:
        persisted_document = docs[0]

    return {
        "action": "verify-go",
        "collection": collection_name,
        "count": count,
        "top_id": top_id,
        "updated_id": updated_id,
        "updated_document": persisted_document,
    }


def create_python_collection(persist_path: str, collection_name: str) -> dict:
    client = chromadb.PersistentClient(path=persist_path)

    try:
        client.delete_collection(collection_name)
    except Exception:
        pass

    collection = client.create_collection(name=collection_name)
    ids = ["py-1", "py-2", "py-3"]
    documents = [
        "python document 1",
        "python document 2",
        "python document 3",
    ]
    embeddings = [
        [1.0, 0.0, 0.0],
        [0.0, 1.0, 0.0],
        [0.0, 0.0, 1.0],
    ]

    collection.add(ids=ids, documents=documents, embeddings=embeddings)

    query = collection.query(
        query_embeddings=[PY_QUERY_EMBEDDING],
        n_results=1,
        include=["documents", "distances"],
    )
    top_id = _extract_top_id(query)

    return {
        "action": "create-python",
        "collection": collection_name,
        "count": collection.count(),
        "top_id": top_id,
        "ids": ids,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="Local persistence cross-check harness")
    parser.add_argument(
        "--action",
        required=True,
        choices=["verify-go", "create-python"],
        help="Harness action to execute",
    )
    parser.add_argument("--persist-path", required=True, help="Persistent Chroma path")
    parser.add_argument("--collection", required=True, help="Collection name")
    args = parser.parse_args()

    try:
        if args.action == "verify-go":
            result = verify_go_collection(args.persist_path, args.collection)
        else:
            result = create_python_collection(args.persist_path, args.collection)

        result["status"] = "success"
        print(json.dumps(result))
        return 0
    except Exception as exc:
        print(
            json.dumps(
                {
                    "status": "error",
                    "action": args.action,
                    "collection": args.collection,
                    "error": str(exc),
                }
            )
        )
        return 1


if __name__ == "__main__":
    sys.exit(main())
