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

package aggfuncs_test

import (
	. "github.com/pingcap/check"
	"github.com/pingcap/parser/ast"
	"github.com/pingcap/parser/mysql"
)

func (s *testSuite) TestMergePartialResult4FirstRow(c *C) {
	tests := []aggTest{
		buildAggTester(ast.AggFuncFirstRow, mysql.TypeLonglong, 5, 0, 2, 0),
		buildAggTester(ast.AggFuncFirstRow, mysql.TypeFloat, 5, 0.0, 2.0, 0.0),
		buildAggTester(ast.AggFuncFirstRow, mysql.TypeDouble, 5, 0.0, 2.0, 0.0),
		buildAggTester(ast.AggFuncFirstRow, mysql.TypeString, 5, "0", "2", "0"),
	}
	for _, test := range tests {
		s.testMergePartialResult(c, test)
	}
}
