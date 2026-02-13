package jsonpath

import (
	"reflect"
	"testing"
)

func intPtr(i int) *int {
	return &i
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    *Query
		wantErr bool
	}{
		{
			name:    "根标识符",
			path:    "$",
			want:    &Query{},
			wantErr: false,
		},
		{
			name: "名称选择器-单引号",
			path: "$['name']",
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{
								Type: NameSelector,
								Name: "name",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "名称选择器-双引号",
			path: `$["name"]`,
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{
								Type: NameSelector,
								Name: "name",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "名称选择器-点表示法",
			path: "$.name",
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{
								Type: NameSelector,
								Name: "name",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "通配符选择器",
			path: "$[*]",
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{
								Type: WildcardSelector,
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "通配符-点表示法",
			path: "$.*",
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{
								Type: WildcardSelector,
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "索引选择器-正数",
			path: "$[0]",
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{
								Type:  IndexSelector,
								Index: 0,
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "索引选择器-负数",
			path: "$[-1]",
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{
								Type:  IndexSelector,
								Index: -1,
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "索引选择器-多个",
			path: "$[0,1,2]",
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{Type: IndexSelector, Index: 0},
							{Type: IndexSelector, Index: 1},
							{Type: IndexSelector, Index: 2},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "数组切片-完整",
			path: "$[0:5:2]",
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{
								Type: SliceSelector,
								Slice: &SliceParams{
									Start: intPtr(0),
									End:   intPtr(5),
									Step:  intPtr(2),
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "数组切片-只有start:end",
			path: "$[1:3]",
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{
								Type: SliceSelector,
								Slice: &SliceParams{
									Start: intPtr(1),
									End:   intPtr(3),
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "数组切片-只有start:",
			path: "$[2:]",
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{
								Type: SliceSelector,
								Slice: &SliceParams{
									Start: intPtr(2),
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "数组切片-只有:end",
			path: "$[:3]",
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{
								Type: SliceSelector,
								Slice: &SliceParams{
									End: intPtr(3),
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "数组切片-负数step",
			path: "$[5:1:-2]",
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{
								Type: SliceSelector,
								Slice: &SliceParams{
									Start: intPtr(5),
									End:   intPtr(1),
									Step:  intPtr(-2),
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "数组切片-全倒序",
			path: "$[::-1]",
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{
								Type: SliceSelector,
								Slice: &SliceParams{
									Step: intPtr(-1),
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "过滤选择器-存在性测试",
			path: "$[?@.foo]",
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{
								Type: FilterSelector,
								Filter: &FilterExpr{
									Type: FilterTest,
									Test: &TestExpr{
										FilterQuery: &FilterQuery{
											Relative: true,
											Segments: []*Segment{
												{
													Type: ChildSegment,
													Selectors: []*Selector{
														{Type: NameSelector, Name: "foo"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "过滤选择器-相等比较",
			path: "$[?@.price == 10]",
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{
								Type: FilterSelector,
								Filter: &FilterExpr{
									Type: FilterComparison,
									Comp: &Comparison{
										Left: &Comparable{
											Type: ComparableSingularQuery,
											SingularQuery: &SingularQuery{
												Relative: true,
												Segments: []*SingularSegment{
													{Type: SingularNameSegment, Name: "price"},
												},
											},
										},
										Op: CompEq,
										Right: &Comparable{
											Type:    ComparableLiteral,
											Literal: &LiteralValue{Type: LiteralNumber, Value: "10"},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "过滤选择器-小于比较",
			path: "$[?@.price < 10]",
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{
								Type: FilterSelector,
								Filter: &FilterExpr{
									Type: FilterComparison,
									Comp: &Comparison{
										Left: &Comparable{
											Type: ComparableSingularQuery,
											SingularQuery: &SingularQuery{
												Relative: true,
												Segments: []*SingularSegment{
													{Type: SingularNameSegment, Name: "price"},
												},
											},
										},
										Op: CompLt,
										Right: &Comparable{
											Type:    ComparableLiteral,
											Literal: &LiteralValue{Type: LiteralNumber, Value: "10"},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "过滤选择器-逻辑与",
			path: "$[?@.price > 5 && @.price < 10]",
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{
								Type: FilterSelector,
								Filter: &FilterExpr{
									Type: FilterLogicalAnd,
									Left: &FilterExpr{
										Type: FilterComparison,
										Comp: &Comparison{
											Left: &Comparable{
												Type: ComparableSingularQuery,
												SingularQuery: &SingularQuery{
													Relative: true,
													Segments: []*SingularSegment{
														{Type: SingularNameSegment, Name: "price"},
													},
												},
											},
											Op: CompGt,
											Right: &Comparable{
												Type:    ComparableLiteral,
												Literal: &LiteralValue{Type: LiteralNumber, Value: "5"},
											},
										},
									},
									Right: &FilterExpr{
										Type: FilterComparison,
										Comp: &Comparison{
											Left: &Comparable{
												Type: ComparableSingularQuery,
												SingularQuery: &SingularQuery{
													Relative: true,
													Segments: []*SingularSegment{
														{Type: SingularNameSegment, Name: "price"},
													},
												},
											},
											Op: CompLt,
											Right: &Comparable{
												Type:    ComparableLiteral,
												Literal: &LiteralValue{Type: LiteralNumber, Value: "10"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "过滤选择器-逻辑或",
			path: "$[?@.price < 5 || @.price > 10]",
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{
								Type: FilterSelector,
								Filter: &FilterExpr{
									Type: FilterLogicalOr,
									Left: &FilterExpr{
										Type: FilterComparison,
										Comp: &Comparison{
											Left: &Comparable{
												Type: ComparableSingularQuery,
												SingularQuery: &SingularQuery{
													Relative: true,
													Segments: []*SingularSegment{
														{Type: SingularNameSegment, Name: "price"},
													},
												},
											},
											Op: CompLt,
											Right: &Comparable{
												Type:    ComparableLiteral,
												Literal: &LiteralValue{Type: LiteralNumber, Value: "5"},
											},
										},
									},
									Right: &FilterExpr{
										Type: FilterComparison,
										Comp: &Comparison{
											Left: &Comparable{
												Type: ComparableSingularQuery,
												SingularQuery: &SingularQuery{
													Relative: true,
													Segments: []*SingularSegment{
														{Type: SingularNameSegment, Name: "price"},
													},
												},
											},
											Op: CompGt,
											Right: &Comparable{
												Type:    ComparableLiteral,
												Literal: &LiteralValue{Type: LiteralNumber, Value: "10"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "过滤选择器-逻辑非",
			path: "$[?!(@.price)]",
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{
								Type: FilterSelector,
								Filter: &FilterExpr{
									Type: FilterLogicalNot,
									Operand: &FilterExpr{
										Type: FilterParen,
										Operand: &FilterExpr{
											Type: FilterTest,
											Test: &TestExpr{
												FilterQuery: &FilterQuery{
													Relative: true,
													Segments: []*Segment{
														{
															Type: ChildSegment,
															Selectors: []*Selector{
																{Type: NameSelector, Name: "price"},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "后代节点-通配符",
			path: "$..*",
			want: &Query{
				Segments: []*Segment{
					{
						Type: DescendantSegment,
						Selectors: []*Selector{
							{Type: WildcardSelector},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "后代节点-名称",
			path: "$..price",
			want: &Query{
				Segments: []*Segment{
					{
						Type: DescendantSegment,
						Selectors: []*Selector{
							{Type: NameSelector, Name: "price"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "后代节点-索引",
			path: "$..[0]",
			want: &Query{
				Segments: []*Segment{
					{
						Type: DescendantSegment,
						Selectors: []*Selector{
							{Type: IndexSelector, Index: 0},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "后代节点-组合",
			path: "$..book[0]",
			want: &Query{
				Segments: []*Segment{
					{
						Type: DescendantSegment,
						Selectors: []*Selector{
							{Type: NameSelector, Name: "book"},
						},
					},
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{Type: IndexSelector, Index: 0},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "嵌套子节点",
			path: "$.store.book[0].author",
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{Type: NameSelector, Name: "store"},
						},
					},
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{Type: NameSelector, Name: "book"},
						},
					},
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{Type: IndexSelector, Index: 0},
						},
					},
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{Type: NameSelector, Name: "author"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "混合选择器-逗号分隔",
			path: "$[0,1,2:4]",
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{Type: IndexSelector, Index: 0},
							{Type: IndexSelector, Index: 1},
							{
								Type: SliceSelector,
								Slice: &SliceParams{
									Start: intPtr(2),
									End:   intPtr(4),
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "混合选择器-名称和通配符",
			path: "$['name','age',*]",
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{Type: NameSelector, Name: "name"},
							{Type: NameSelector, Name: "age"},
							{Type: WildcardSelector},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "字符串转义-单引号内双引号",
			path: `$['a"b']`,
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{Type: NameSelector, Name: `a"b`},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "字符串转义-双引号内单引号",
			path: `$["a'b"]`,
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{Type: NameSelector, Name: "a'b"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "字符串转义-反斜杠转义",
			path: `$['a\n']`,
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{Type: NameSelector, Name: "a\n"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "字符串转义-Unicode转义",
			path: `$['a\u0041']`,
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{Type: NameSelector, Name: "aA"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "函数-length()",
			path: "$[?length(@.name) > 5]",
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{
								Type: FilterSelector,
								Filter: &FilterExpr{
									Type: FilterComparison,
									Comp: &Comparison{
										Left: &Comparable{
											Type: ComparableFuncExpr,
											FuncExpr: &FuncCall{
												Name: "length",
												Args: []*FuncArg{
													{
														Type: FuncArgFilterQuery,
														FilterQuery: &FilterQuery{
															Relative: true,
															Segments: []*Segment{
																{
																	Type: ChildSegment,
																	Selectors: []*Selector{
																		{Type: NameSelector, Name: "name"},
																	},
																},
															},
														},
													},
												},
											},
										},
										Op: CompGt,
										Right: &Comparable{
											Type:    ComparableLiteral,
											Literal: &LiteralValue{Type: LiteralNumber, Value: "5"},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "函数-count()",
			path: "$[?count(@.*) > 1]",
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{
								Type: FilterSelector,
								Filter: &FilterExpr{
									Type: FilterComparison,
									Comp: &Comparison{
										Left: &Comparable{
											Type: ComparableFuncExpr,
											FuncExpr: &FuncCall{
												Name: "count",
												Args: []*FuncArg{
													{
														Type: FuncArgFilterQuery,
														FilterQuery: &FilterQuery{
															Relative: true,
															Segments: []*Segment{
																{
																	Type: ChildSegment,
																	Selectors: []*Selector{
																		{Type: WildcardSelector},
																	},
																},
															},
														},
													},
												},
											},
										},
										Op: CompGt,
										Right: &Comparable{
											Type:    ComparableLiteral,
											Literal: &LiteralValue{Type: LiteralNumber, Value: "1"},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "函数-match()",
			path: `$[?match(@.date, "1974-05-..")]`,
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{
								Type: FilterSelector,
								Filter: &FilterExpr{
									Type: FilterTest,
									Test: &TestExpr{
										FuncExpr: &FuncCall{
											Name: "match",
											Args: []*FuncArg{
												{
													Type: FuncArgFilterQuery,
													FilterQuery: &FilterQuery{
														Relative: true,
														Segments: []*Segment{
															{
																Type: ChildSegment,
																Selectors: []*Selector{
																	{Type: NameSelector, Name: "date"},
																},
															},
														},
													},
												},
												{
													Type:    FuncArgLiteral,
													Literal: &LiteralValue{Type: LiteralString, Value: "1974-05-.."},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "函数-search()",
			path: `$[?search(@.author, "[BR]ob")]`,
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{
								Type: FilterSelector,
								Filter: &FilterExpr{
									Type: FilterTest,
									Test: &TestExpr{
										FuncExpr: &FuncCall{
											Name: "search",
											Args: []*FuncArg{
												{
													Type: FuncArgFilterQuery,
													FilterQuery: &FilterQuery{
														Relative: true,
														Segments: []*Segment{
															{
																Type: ChildSegment,
																Selectors: []*Selector{
																	{Type: NameSelector, Name: "author"},
																},
															},
														},
													},
												},
												{
													Type:    FuncArgLiteral,
													Literal: &LiteralValue{Type: LiteralString, Value: "[BR]ob"},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "函数-value()",
			path: "$[?value(@..color) == 'red']",
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{
								Type: FilterSelector,
								Filter: &FilterExpr{
									Type: FilterComparison,
									Comp: &Comparison{
										Left: &Comparable{
											Type: ComparableFuncExpr,
											FuncExpr: &FuncCall{
												Name: "value",
												Args: []*FuncArg{
													{
														Type: FuncArgFilterQuery,
														FilterQuery: &FilterQuery{
															Relative: true,
															Segments: []*Segment{
																{
																	Type: DescendantSegment,
																	Selectors: []*Selector{
																		{Type: NameSelector, Name: "color"},
																	},
																},
															},
														},
													},
												},
											},
										},
										Op: CompEq,
										Right: &Comparable{
											Type:    ComparableLiteral,
											Literal: &LiteralValue{Type: LiteralString, Value: "red"},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "当前节点标识符-在过滤中",
			path: "$[?@.foo == @.bar]",
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{
								Type: FilterSelector,
								Filter: &FilterExpr{
									Type: FilterComparison,
									Comp: &Comparison{
										Left: &Comparable{
											Type: ComparableSingularQuery,
											SingularQuery: &SingularQuery{
												Relative: true,
												Segments: []*SingularSegment{
													{Type: SingularNameSegment, Name: "foo"},
												},
											},
										},
										Op: CompEq,
										Right: &Comparable{
											Type: ComparableSingularQuery,
											SingularQuery: &SingularQuery{
												Relative: true,
												Segments: []*SingularSegment{
													{Type: SingularNameSegment, Name: "bar"},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "嵌套过滤",
			path: "$.a[?@.b[?@.c == 'x']]",
			want: &Query{
				Segments: []*Segment{
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{Type: NameSelector, Name: "a"},
						},
					},
					{
						Type: ChildSegment,
						Selectors: []*Selector{
							{
								Type: FilterSelector,
								Filter: &FilterExpr{
									Type: FilterTest,
									Test: &TestExpr{
										FilterQuery: &FilterQuery{
											Relative: true,
											Segments: []*Segment{
												{
													Type: ChildSegment,
													Selectors: []*Selector{
														{Type: NameSelector, Name: "b"},
													},
												},
												{
													Type: ChildSegment,
													Selectors: []*Selector{
														{
															Type: FilterSelector,
															Filter: &FilterExpr{
																Type: FilterComparison,
																Comp: &Comparison{
																	Left: &Comparable{
																		Type: ComparableSingularQuery,
																		SingularQuery: &SingularQuery{
																			Relative: true,
																			Segments: []*SingularSegment{
																				{Type: SingularNameSegment, Name: "c"},
																			},
																		},
																	},
																	Op: CompEq,
																	Right: &Comparable{
																		Type:    ComparableLiteral,
																		Literal: &LiteralValue{Type: LiteralString, Value: "x"},
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := Parse(tt.path)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Parse() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Parse() succeeded unexpectedly")
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkParse(b *testing.B) {
	benchmarks := []struct {
		name string
		path string
	}{
		// 基础场景
		{"根节点", "$"},
		{"简单属性", "$.name"},
		{"嵌套属性", "$.store.book[0].author"},
		{"数组索引", "$[0]"},
		{"多索引", "$[0,1,2]"},
		{"数组切片", "$[0:5:2]"},
		{"倒序切片", "$[::-1]"},

		// 通配符场景
		{"通配符", "$.*"},
		{"数组通配符", "$[*]"},
		{"后代通配符", "$..*"},
		{"后代属性", "$..book"},

		// 过滤器场景
		{"简单过滤-存在性", "$[?@.foo]"},
		{"简单过滤-比较", "$[?@.price == 10]"},
		{"过滤器-小于", "$[?@.price < 10]"},
		{"过滤器-大于", "$[?@.price > 5]"},

		// 逻辑运算
		{"逻辑与", "$[?@.price > 5 && @.price < 10]"},
		{"逻辑或", "$[?@.price < 5 || @.price > 10]"},
		{"逻辑非", "$[?!(@.price)]"},

		// 函数调用
		{"函数-length", "$[?length(@.authors) >= 5]"},
		{"函数-count", "$[?count(@.*.author) >= 5]"},
		{"函数-match", `$[?match(@.date, "1974-05-..")]`},
		{"函数-search", `$[?search(@.author, "[BR]ob")]`},
		{"函数-value", `$[?value(@..color) == "red"]`},

		// 复杂场景
		{"嵌套过滤", "$.a[?@.b[?@.c == 'x']]"},
		{"混合选择器", "$[0,1,2:4]"},
		{"当前节点比较", "$[?@.foo == @.bar]"},
		{"深度嵌套", "$.store.book[0].authors[0].name"},
		{"后代+索引", "$..book[0]"},
		{"后代+过滤", "$..book[?@.price < 10]"},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = Parse(bm.path)
			}
		})
	}
}
