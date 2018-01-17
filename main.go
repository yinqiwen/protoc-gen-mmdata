package main

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/golang/protobuf/proto"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
)

func main() {

	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("reading input:%v", err)
	}

	var request plugin.CodeGeneratorRequest // The input.
	//var response plugin.CodeGeneratorResponse // The output.
	if err := proto.Unmarshal(data, &request); err != nil {
		log.Fatalf("parsing input proto:%v", err)
	}

	if len(request.FileToGenerate) == 0 {
		log.Fatalf("no files to generate")
	}

	for _, file := range request.ProtoFile {
		g := &Generator{}
		if !g.Verify(file) {
			continue
		}
		g.BuildTypeNameMap(file)
		g.DumpHeader(file.GetName())
		tab, tabs := g.DumpNamespaceBegin(*file.Package)

		for _, msg := range file.MessageType {
			g.DumpMessage(msg, tab)
		}
		g.DumpNamespaceEnd(tabs)
		g.Finish()
		g.DumpFile()
	}

	//log.Printf("\n%s", g.OutputBuffer.String())
}
