package chroma

import (
	"fmt"
	"github.com/owulveryck/onnx-go"
	"github.com/owulveryck/onnx-go/backend/x/gorgonnx"
	"gorgonia.org/tensor"
	"io/ioutil"
	"log"
	"testing"
)

func stringToFloat32(s string) float32 {
	// Implement your conversion logic here
	return 0.1 // Placeholder
}

func Test_onnx(t *testing.T) {
	// Create a backend receiver
	backend := gorgonnx.NewGraph()
	// Create a model and set the execution backend
	model := onnx.NewModel(backend)

	// read the onnx model
	b, _ := ioutil.ReadFile("/Users/tazarov/.cache/chroma/onnx_models/all-MiniLM-L6-v2/onnx/model.onnx")
	// Decode it into the model
	err := model.UnmarshalBinary(b)
	if err != nil {
		log.Fatal(err)
	}
	// Set the first input, the number depends of the model
	var stringList = []string{"string1", "string2", "string3"}
	var floatList = make([]float32, len(stringList))
	for i, s := range stringList {
		floatList[i] = stringToFloat32(s)
	}

	// Create a tensor from the float slice
	inputTensor := tensor.New(
		tensor.WithShape(len(floatList)), // Set the appropriate shape
		tensor.Of(tensor.Float32),        // Set the tensor data type
		tensor.WithBacking(floatList),    // Use the float slice as the backing data
	)
	fmt.Println(inputTensor.Data())
	err = model.SetInput(0, inputTensor)

	fmt.Println("--2232-test")
	if err != nil {
		log.Fatal(err)
	}
	err = backend.Run()
	if err != nil {
		log.Fatal(err)
	}
	// Check error
	output, _ := model.GetOutputTensors()
	// write the first output to stdout

	fmt.Println(output[0])
}
