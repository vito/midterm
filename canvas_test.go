package midterm_test

import (
	"testing"

	"github.com/muesli/termenv"
	"github.com/stretchr/testify/require"
	"github.com/vito/midterm"
)

var red = midterm.Format{Fg: termenv.ANSIRed}
var green = midterm.Format{Fg: termenv.ANSIGreen}
var blue = midterm.Format{Fg: termenv.ANSIBlue}
var brightGreen = midterm.Format{Fg: termenv.ANSIBrightGreen}

func TestCanvasPaint(t *testing.T) {
	type PaintExample struct {
		Name   string
		Paints []midterm.Cursor
		Result *midterm.Canvas
	}

	for _, ex := range []PaintExample{
		{
			Name: "initial paint",
			Paints: []midterm.Cursor{
				{0, 0, midterm.EmptyFormat},
			},
			Result: &midterm.Canvas{
				Rows: []*midterm.Region{
					{
						F:    midterm.EmptyFormat,
						Size: 1,
						Next: nil,
					},
				},
			},
		},
		{
			Name: "repeated paints does not grow",
			Paints: []midterm.Cursor{
				{0, 0, midterm.EmptyFormat},
				{0, 0, midterm.EmptyFormat},
				{0, 0, midterm.EmptyFormat},
			},
			Result: &midterm.Canvas{
				Rows: []*midterm.Region{
					{
						F:    midterm.EmptyFormat,
						Size: 1,
						Next: nil,
					},
				},
			},
		},
		{
			Name: "painting on boundary grows",
			Paints: []midterm.Cursor{
				{0, 0, midterm.EmptyFormat},
				{0, 1, midterm.EmptyFormat},
				{0, 2, midterm.EmptyFormat},
			},
			Result: &midterm.Canvas{
				Rows: []*midterm.Region{
					{
						F:    midterm.EmptyFormat,
						Size: 3,
						Next: nil,
					},
				},
			},
		},
		{
			Name: "within region with same format does nothing",
			Paints: []midterm.Cursor{
				{0, 0, red},
				{0, 1, red},
				{0, 2, red},
				{0, 1, red},
			},
			Result: &midterm.Canvas{
				Rows: []*midterm.Region{
					{
						F:    red,
						Size: 3,
						Next: nil,
					},
				},
			},
		},
		{
			Name: "within region with different format splits the region",
			Paints: []midterm.Cursor{
				{0, 0, red},
				{0, 1, red},
				{0, 2, red},
				{0, 3, red},
				{0, 1, green},
			},
			Result: &midterm.Canvas{
				Rows: []*midterm.Region{
					{
						F:    red,
						Size: 1,
						Next: &midterm.Region{
							F:    green,
							Size: 1,
							Next: &midterm.Region{
								F:    red,
								Size: 2,
							},
						},
					},
				},
			},
		},
		{
			Name: "different format between regions at start of next region",
			Paints: []midterm.Cursor{
				{0, 0, red},
				{0, 1, red},
				{0, 2, red},
				{0, 3, green},
				{0, 4, green},
				{0, 5, green},
				{0, 3, blue},
			},
			Result: &midterm.Canvas{
				Rows: []*midterm.Region{
					{
						F:    red,
						Size: 3,
						Next: &midterm.Region{
							F:    blue,
							Size: 1,
							Next: &midterm.Region{
								F:    green,
								Size: 2,
							},
						},
					},
				},
			},
		},
		{
			Name: "same format between regions at start of next region",
			Paints: []midterm.Cursor{
				{0, 0, red},
				{0, 1, red},
				{0, 2, red},
				{0, 3, green},
				{0, 4, green},
				{0, 5, green},
				{0, 3, green},
			},
			Result: &midterm.Canvas{
				Rows: []*midterm.Region{
					{
						F:    red,
						Size: 3,
						Next: &midterm.Region{
							F:    green,
							Size: 3,
						},
					},
				},
			},
		},
		{
			Name: "painting beyond the end of the region",
			Paints: []midterm.Cursor{
				{0, 0, red},
				{0, 1, red},
				{0, 2, red},
				{0, 4, green},
				{0, 5, green},
			},
			Result: &midterm.Canvas{
				Rows: []*midterm.Region{
					{
						F:    red,
						Size: 3,
						Next: &midterm.Region{
							F:    midterm.EmptyFormat,
							Size: 1,
							Next: &midterm.Region{
								F:    green,
								Size: 2,
							},
						},
					},
				},
			},
		},
		{
			Name: "overwriting a single width region",
			Paints: []midterm.Cursor{
				{0, 0, red},
				{0, 1, red},
				{0, 2, green},
				{0, 3, blue},
				{0, 2, brightGreen},
			},
			Result: &midterm.Canvas{
				Rows: []*midterm.Region{
					{
						F:    red,
						Size: 2,
						Next: &midterm.Region{
							F:    brightGreen,
							Size: 1,
							Next: &midterm.Region{
								F:    blue,
								Size: 1,
							},
						},
					},
				},
			},
		},
		{
			Name: "overwriting the start of a wider region",
			Paints: []midterm.Cursor{
				{0, 0, red},
				{0, 1, red},
				{0, 2, green},
				{0, 3, green},
				{0, 2, brightGreen},
			},
			Result: &midterm.Canvas{
				Rows: []*midterm.Region{
					{
						F:    red,
						Size: 2,
						Next: &midterm.Region{
							F:    brightGreen,
							Size: 1,
							Next: &midterm.Region{
								F:    green,
								Size: 1,
							},
						},
					},
				},
			},
		},
		{
			Name: "overwriting the end of a wider region",
			Paints: []midterm.Cursor{
				{0, 0, red},
				{0, 1, red},
				{0, 2, green},
				{0, 3, green},
				{0, 4, green},
				{0, 4, brightGreen},
			},
			Result: &midterm.Canvas{
				Rows: []*midterm.Region{
					{
						F:    red,
						Size: 2,
						Next: &midterm.Region{
							F:    green,
							Size: 2,
							Next: &midterm.Region{
								F:    brightGreen,
								Size: 1,
							},
						},
					},
				},
			},
		},
		{
			Name: "overwriting the end of a wider region with another region after it",
			Paints: []midterm.Cursor{
				{0, 0, red},
				{0, 1, red},
				{0, 2, green},
				{0, 3, green},
				{0, 4, green},
				{0, 5, blue},
				{0, 6, blue},
				{0, 4, brightGreen},
			},
			Result: &midterm.Canvas{
				Rows: []*midterm.Region{
					{
						F:    red,
						Size: 2,
						Next: &midterm.Region{
							F:    green,
							Size: 2,
							Next: &midterm.Region{
								F:    brightGreen,
								Size: 1,
								Next: &midterm.Region{
									F:    blue,
									Size: 2,
								},
							},
						},
					},
				},
			},
		},
		{
			Name: "clobbering a region at the start",
			Paints: []midterm.Cursor{
				{0, 0, red},
				{0, 1, green},
				{0, 0, blue},
			},
			Result: &midterm.Canvas{
				Rows: []*midterm.Region{
					{
						F:    blue,
						Size: 1,
						Next: &midterm.Region{
							F:    green,
							Size: 1,
						},
					},
				},
			},
		},
		{
			Name: "clobbering a region at the end",
			Paints: []midterm.Cursor{
				{0, 0, red},
				{0, 1, green},
				{0, 1, blue},
			},
			Result: &midterm.Canvas{
				Rows: []*midterm.Region{
					{
						F:    red,
						Size: 1,
						Next: &midterm.Region{
							F:    blue,
							Size: 1,
						},
					},
				},
			},
		},
		{
			Name: "overwriting the middle of a wider region",
			Paints: []midterm.Cursor{
				{0, 0, red},
				{0, 1, red},
				{0, 2, green},
				{0, 3, green},
				{0, 4, green},
				{0, 3, brightGreen},
			},
			Result: &midterm.Canvas{
				Rows: []*midterm.Region{
					{
						F:    red,
						Size: 2,
						Next: &midterm.Region{
							F:    green,
							Size: 1,
							Next: &midterm.Region{
								F:    brightGreen,
								Size: 1,
								Next: &midterm.Region{
									F:    green,
									Size: 1,
								},
							},
						},
					},
				},
			},
		},
	} {
		t.Run(ex.Name, func(t *testing.T) {
			canvas := &midterm.Canvas{}
			for _, p := range ex.Paints {
				t.Logf("painting %+v", p)
				canvas.Paint(p)
			}
			require.Equal(t, ex.Result, canvas)
		})
	}
}

// func TestCanvasInsert(t *testing.T) {
// 	red := midterm.Format{Fg: termenv.ANSIRed}
// 	green := midterm.Format{Fg: termenv.ANSIGreen}
// 	blue := midterm.Format{Fg: termenv.ANSIBlue}
// 	brightGreen := midterm.Format{Fg: termenv.ANSIBrightGreen}
//
// 	type PaintExample struct {
// 		Name   string
// 		Paints []midterm.Cursor
// 		Insert midterm.Cursor
// 		Size   int
// 		Result *midterm.Canvas
// 	}
//
// 	for _, ex := range []PaintExample{
// 		{
// 			Name: "initial paint",
// 			Paints: []midterm.Cursor{
// 				{0, 0, midterm.EmptyFormat},
// 			},
// 			Result: &midterm.Canvas{
// 				Rows: []*midterm.Region{
// 					{
// 						F:    midterm.EmptyFormat,
// 						Size: 1,
// 						Next: nil,
// 					},
// 				},
// 			},
// 		},
// 		{
// 			Name: "repeated paints does not grow",
// 			Paints: []midterm.Cursor{
// 				{0, 0, midterm.EmptyFormat},
// 				{0, 0, midterm.EmptyFormat},
// 				{0, 0, midterm.EmptyFormat},
// 			},
// 			Result: &midterm.Canvas{
// 				Rows: []*midterm.Region{
// 					{
// 						F:    midterm.EmptyFormat,
// 						Size: 1,
// 						Next: nil,
// 					},
// 				},
// 			},
// 		},
// 		{
// 			Name: "painting on boundary grows",
// 			Paints: []midterm.Cursor{
// 				{0, 0, midterm.EmptyFormat},
// 				{0, 1, midterm.EmptyFormat},
// 				{0, 2, midterm.EmptyFormat},
// 			},
// 			Result: &midterm.Canvas{
// 				Rows: []*midterm.Region{
// 					{
// 						F:    midterm.EmptyFormat,
// 						Size: 3,
// 						Next: nil,
// 					},
// 				},
// 			},
// 		},
// 		{
// 			Name: "within region with same format does nothing",
// 			Paints: []midterm.Cursor{
// 				{0, 0, red},
// 				{0, 1, red},
// 				{0, 2, red},
// 				{0, 1, red},
// 			},
// 			Result: &midterm.Canvas{
// 				Rows: []*midterm.Region{
// 					{
// 						F:    red,
// 						Size: 3,
// 						Next: nil,
// 					},
// 				},
// 			},
// 		},
// 		{
// 			Name: "within region with different format splits the region",
// 			Paints: []midterm.Cursor{
// 				{0, 0, red},
// 				{0, 1, red},
// 				{0, 2, red},
// 				{0, 3, red},
// 				{0, 1, green},
// 			},
// 			Result: &midterm.Canvas{
// 				Rows: []*midterm.Region{
// 					{
// 						F:    red,
// 						Size: 1,
// 						Next: &midterm.Region{
// 							F:    green,
// 							Size: 1,
// 							Next: &midterm.Region{
// 								F:    red,
// 								Size: 2,
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 		{
// 			Name: "different format between regions at start of next region",
// 			Paints: []midterm.Cursor{
// 				{0, 0, red},
// 				{0, 1, red},
// 				{0, 2, red},
// 				{0, 3, green},
// 				{0, 4, green},
// 				{0, 5, green},
// 				{0, 3, blue},
// 			},
// 			Result: &midterm.Canvas{
// 				Rows: []*midterm.Region{
// 					{
// 						F:    red,
// 						Size: 3,
// 						Next: &midterm.Region{
// 							F:    blue,
// 							Size: 1,
// 							Next: &midterm.Region{
// 								F:    green,
// 								Size: 2,
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 		{
// 			Name: "same format between regions at start of next region",
// 			Paints: []midterm.Cursor{
// 				{0, 0, red},
// 				{0, 1, red},
// 				{0, 2, red},
// 				{0, 3, green},
// 				{0, 4, green},
// 				{0, 5, green},
// 				{0, 3, green},
// 			},
// 			Result: &midterm.Canvas{
// 				Rows: []*midterm.Region{
// 					{
// 						F:    red,
// 						Size: 3,
// 						Next: &midterm.Region{
// 							F:    green,
// 							Size: 3,
// 						},
// 					},
// 				},
// 			},
// 		},
// 		{
// 			Name: "painting beyond the end of the region",
// 			Paints: []midterm.Cursor{
// 				{0, 0, red},
// 				{0, 1, red},
// 				{0, 2, red},
// 				{0, 4, green},
// 				{0, 5, green},
// 			},
// 			Result: &midterm.Canvas{
// 				Rows: []*midterm.Region{
// 					{
// 						F:    red,
// 						Size: 3,
// 						Next: &midterm.Region{
// 							F:    midterm.EmptyFormat,
// 							Size: 1,
// 							Next: &midterm.Region{
// 								F:    green,
// 								Size: 2,
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 		{
// 			Name: "overwriting a single width region",
// 			Paints: []midterm.Cursor{
// 				{0, 0, red},
// 				{0, 1, red},
// 				{0, 2, green},
// 				{0, 3, blue},
// 				{0, 2, brightGreen},
// 			},
// 			Result: &midterm.Canvas{
// 				Rows: []*midterm.Region{
// 					{
// 						F:    red,
// 						Size: 2,
// 						Next: &midterm.Region{
// 							F:    brightGreen,
// 							Size: 1,
// 							Next: &midterm.Region{
// 								F:    blue,
// 								Size: 1,
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 		{
// 			Name: "overwriting the start of a wider region",
// 			Paints: []midterm.Cursor{
// 				{0, 0, red},
// 				{0, 1, red},
// 				{0, 2, green},
// 				{0, 3, green},
// 				{0, 2, brightGreen},
// 			},
// 			Result: &midterm.Canvas{
// 				Rows: []*midterm.Region{
// 					{
// 						F:    red,
// 						Size: 2,
// 						Next: &midterm.Region{
// 							F:    brightGreen,
// 							Size: 1,
// 							Next: &midterm.Region{
// 								F:    green,
// 								Size: 1,
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 		{
// 			Name: "overwriting the end of a wider region",
// 			Paints: []midterm.Cursor{
// 				{0, 0, red},
// 				{0, 1, red},
// 				{0, 2, green},
// 				{0, 3, green},
// 				{0, 4, green},
// 				{0, 4, brightGreen},
// 			},
// 			Result: &midterm.Canvas{
// 				Rows: []*midterm.Region{
// 					{
// 						F:    red,
// 						Size: 2,
// 						Next: &midterm.Region{
// 							F:    green,
// 							Size: 2,
// 							Next: &midterm.Region{
// 								F:    brightGreen,
// 								Size: 1,
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 		{
// 			Name: "overwriting the end of a wider region with another region after it",
// 			Paints: []midterm.Cursor{
// 				{0, 0, red},
// 				{0, 1, red},
// 				{0, 2, green},
// 				{0, 3, green},
// 				{0, 4, green},
// 				{0, 5, blue},
// 				{0, 6, blue},
// 				{0, 4, brightGreen},
// 			},
// 			Result: &midterm.Canvas{
// 				Rows: []*midterm.Region{
// 					{
// 						F:    red,
// 						Size: 2,
// 						Next: &midterm.Region{
// 							F:    green,
// 							Size: 2,
// 							Next: &midterm.Region{
// 								F:    brightGreen,
// 								Size: 1,
// 								Next: &midterm.Region{
// 									F:    blue,
// 									Size: 2,
// 								},
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 		{
// 			Name: "clobbering a region at the start",
// 			Paints: []midterm.Cursor{
// 				{0, 0, red},
// 				{0, 1, green},
// 				{0, 0, blue},
// 			},
// 			Result: &midterm.Canvas{
// 				Rows: []*midterm.Region{
// 					{
// 						F:    blue,
// 						Size: 1,
// 						Next: &midterm.Region{
// 							F:    green,
// 							Size: 1,
// 						},
// 					},
// 				},
// 			},
// 		},
// 		{
// 			Name: "clobbering a region at the end",
// 			Paints: []midterm.Cursor{
// 				{0, 0, red},
// 				{0, 1, green},
// 				{0, 1, blue},
// 			},
// 			Result: &midterm.Canvas{
// 				Rows: []*midterm.Region{
// 					{
// 						F:    red,
// 						Size: 1,
// 						Next: &midterm.Region{
// 							F:    blue,
// 							Size: 1,
// 						},
// 					},
// 				},
// 			},
// 		},
// 		{
// 			Name: "overwriting the middle of a wider region",
// 			Paints: []midterm.Cursor{
// 				{0, 0, red},
// 				{0, 1, red},
// 				{0, 2, green},
// 				{0, 3, green},
// 				{0, 4, green},
// 				{0, 3, brightGreen},
// 			},
// 			Result: &midterm.Canvas{
// 				Rows: []*midterm.Region{
// 					{
// 						F:    red,
// 						Size: 2,
// 						Next: &midterm.Region{
// 							F:    green,
// 							Size: 1,
// 							Next: &midterm.Region{
// 								F:    brightGreen,
// 								Size: 1,
// 								Next: &midterm.Region{
// 									F:    green,
// 									Size: 1,
// 								},
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 	} {
// 		t.Run(ex.Name, func(t *testing.T) {
// 			canvas := &midterm.Canvas{}
// 			for _, p := range ex.Paints {
// 				t.Logf("inserting %+v", p)
// 				canvas.Insert(p)
// 			}
// 			require.Equal(t, ex.Result, canvas)
// 		})
// 	}
// }
