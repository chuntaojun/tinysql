// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

// +build ignore

package main

import (
	"bytes"
	"flag"
	"go/format"
	"io/ioutil"
	"log"
	"path/filepath"
	"text/template"

	. "github.com/pingcap/tidb/expression/generator/helper"
)

const header = `// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by go generate in expression/generator; DO NOT EDIT.

package expression
`

const newLine = "\n"

const builtinCompareImports = `import (
	"github.com/pingcap/tidb/types"
	"github.com/pingcap/tidb/util/chunk"
)
`

var builtinCompareVecTpl = template.Must(template.New("").Parse(`
func (b *builtin{{ .compare.CompareName }}{{ .type.TypeName }}Sig) vecEvalInt(input *chunk.Chunk, result *chunk.Column) error {
	n := input.NumRows()
	buf0, err := b.bufAllocator.get(types.ET{{ .type.ETName }}, n)
	if err != nil {
		return err
	}
	defer b.bufAllocator.put(buf0)
	if err := b.args[0].VecEval{{ .type.TypeName }}(b.ctx, input, buf0); err != nil {
		return err
	}
	buf1, err := b.bufAllocator.get(types.ET{{ .type.ETName }}, n)
	if err != nil {
		return err
	}
	defer b.bufAllocator.put(buf1)
	if err := b.args[1].VecEval{{ .type.TypeName }}(b.ctx, input, buf1); err != nil {
		return err
	}

{{ if .type.Fixed }}
	arg0 := buf0.{{ .type.TypeNameInColumn }}s()
	arg1 := buf1.{{ .type.TypeNameInColumn }}s()
{{- end }}
	result.ResizeInt64(n, false)
	result.MergeNulls(buf0, buf1)
	i64s := result.Int64s()
	for i := 0; i < n; i++ {
		if result.IsNull(i) {
			continue
		}
{{- if eq .type.ETName "Real" }}
		val := types.CompareFloat64(arg0[i], arg1[i])
{{- else }}
		val := types.CompareString(buf0.GetString(i), buf1.GetString(i))
{{- end }}
		if val {{ .compare.Operator }} 0 {
			i64s[i] = 1
		} else {
			i64s[i] = 0
		}
	}
	return nil
}

func (b *builtin{{ .compare.CompareName }}{{ .type.TypeName }}Sig) vectorized() bool {
	return true
}
`))

const builtinCompareVecTestHeader = `import (
	"testing"

	. "github.com/pingcap/check"
	"github.com/pingcap/parser/ast"
	"github.com/pingcap/tidb/types"
)

var vecGeneratedBuiltinCompareCases = map[string][]vecExprBenchCase{
`

var builtinCompareVecTestFuncHeader = template.Must(template.New("").Parse(`	ast.{{ .CompareName }}: {
`))

var builtinCompareVecTestCase = template.Must(template.New("").Parse(`		{retEvalType: types.ETInt, childrenTypes: []types.EvalType{types.ET{{ .ETName }}, types.ET{{ .ETName }}}},
`))

var builtinCompareVecTestFuncTail = `	},
`

var builtinCompareVecTestTail = `}

func (s *testEvaluatorSuite) TestVectorizedGeneratedBuiltinCompareEvalOneVec(c *C) {
	testVectorizedEvalOneVec(c, vecGeneratedBuiltinCompareCases)
}

func (s *testEvaluatorSuite) TestVectorizedGeneratedBuiltinCompareFunc(c *C) {
	testVectorizedBuiltinFunc(c, vecGeneratedBuiltinCompareCases)
}

func BenchmarkVectorizedGeneratedBuiltinCompareEvalOneVec(b *testing.B) {
	benchmarkVectorizedEvalOneVec(b, vecGeneratedBuiltinCompareCases)
}

func BenchmarkVectorizedGeneratedBuiltinCompareFunc(b *testing.B) {
	benchmarkVectorizedBuiltinFunc(b, vecGeneratedBuiltinCompareCases)
}
`

type CompareContext struct {
	// Describe the name of CompareContext(LT/LE/GT/GE/EQ/NE/NullEQ)
	CompareName string
	// Compare Operators
	Operator string
}

var comparesMap = []CompareContext{
	{CompareName: "LT", Operator: "<"},
	{CompareName: "LE", Operator: "<="},
	{CompareName: "GT", Operator: ">"},
	{CompareName: "GE", Operator: ">="},
	{CompareName: "EQ", Operator: "=="},
	{CompareName: "NE", Operator: "!="},
}

var typesMap = []TypeContext{
	TypeInt,
	TypeReal,
	TypeString,
}

func generateDotGo(fileName string, compares []CompareContext, types []TypeContext) (err error) {
	w := new(bytes.Buffer)
	w.WriteString(header)
	w.WriteString(newLine)
	w.WriteString(builtinCompareImports)

	var ctx = make(map[string]interface{})
	for _, compareCtx := range compares {
		for _, typeCtx := range types {
			ctx["compare"] = compareCtx
			ctx["type"] = typeCtx
			if typeCtx.TypeName == TypeInt.TypeName {
				continue
			}
			err := builtinCompareVecTpl.Execute(w, ctx)
			if err != nil {
				return err
			}
		}
	}
	data, err := format.Source(w.Bytes())
	if err != nil {
		log.Println("[Warn]", fileName+": gofmt failed", err)
		data = w.Bytes() // write original data for debugging
	}
	return ioutil.WriteFile(fileName, data, 0644)
}

func generateTestDotGo(fileName string, compares []CompareContext, types []TypeContext) error {
	w := new(bytes.Buffer)
	w.WriteString(header)
	w.WriteString(builtinCompareVecTestHeader)

	for _, compareCtx := range compares {
		err := builtinCompareVecTestFuncHeader.Execute(w, compareCtx)
		if err != nil {
			return err
		}
		for _, typeCtx := range types {
			if typeCtx.TypeName == TypeInt.TypeName {
				continue
			}
			err := builtinCompareVecTestCase.Execute(w, typeCtx)
			if err != nil {
				return err
			}
		}
		w.WriteString(builtinCompareVecTestFuncTail)
	}
	w.WriteString(builtinCompareVecTestTail)

	data, err := format.Source(w.Bytes())
	if err != nil {
		log.Println("[Warn]", fileName+": gofmt failed", err)
		data = w.Bytes() // write original data for debugging
	}
	return ioutil.WriteFile(fileName, data, 0644)
}

// generateOneFile generate one xxx.go file and the associated xxx_test.go file.
func generateOneFile(fileNamePrefix string, compares []CompareContext,
	types []TypeContext) (err error) {

	err = generateDotGo(fileNamePrefix+".go", compares, types)
	if err != nil {
		return
	}
	err = generateTestDotGo(fileNamePrefix+"_test.go", compares, types)
	return
}

func main() {
	flag.Parse()
	var err error
	outputDir := "."
	err = generateOneFile(filepath.Join(outputDir, "builtin_compare_vec_generated"),
		comparesMap, typesMap)
	if err != nil {
		log.Fatalln("generateOneFile", err)
	}
}
