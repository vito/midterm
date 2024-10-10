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
	type paint struct {
		row, col int
		f        midterm.Format
	}

	type PaintExample struct {
		Name   string
		Paints []paint
		Result *midterm.Canvas
	}

	for _, ex := range []PaintExample{
		{
			Name: "initial paint",
			Paints: []paint{
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
			Name: "painting past end paints empty gap",
			Paints: []paint{
				{0, 3, red},
			},
			Result: &midterm.Canvas{
				Rows: []*midterm.Region{
					{
						F:    midterm.EmptyFormat,
						Size: 3,
						Next: &midterm.Region{
							F:    red,
							Size: 1,
						},
					},
				},
			},
		},
		{
			Name: "repeated paints does not grow",
			Paints: []paint{
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
			Paints: []paint{
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
			Paints: []paint{
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
			Paints: []paint{
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
			Paints: []paint{
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
			Paints: []paint{
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
			Paints: []paint{
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
			Paints: []paint{
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
			Paints: []paint{
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
			Paints: []paint{
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
			Paints: []paint{
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
			Paints: []paint{
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
			Paints: []paint{
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
			Paints: []paint{
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
				canvas.Paint(p.row, p.col, p.f)
			}
			require.Equal(t, ex.Result, canvas)
		})
	}
}

func TestCanvasInsert(t *testing.T) {
	type paint struct {
		row, col int
		f        midterm.Format
		insert   int
	}

	type PaintExample struct {
		Name   string
		Paints []paint
		Result *midterm.Canvas
	}

	for _, ex := range []PaintExample{
		{
			Name: "initial insert",
			Paints: []paint{
				{0, 0, midterm.EmptyFormat, 2},
			},
			Result: &midterm.Canvas{
				Rows: []*midterm.Region{
					{
						F:    midterm.EmptyFormat,
						Size: 2,
						Next: nil,
					},
				},
			},
		},
		{
			Name: "repeated inserts at start grow",
			Paints: []paint{
				{0, 0, red, 1},
				{0, 0, red, 1},
				{0, 0, red, 1},
			},
			Result: &midterm.Canvas{
				Rows: []*midterm.Region{
					{
						F:    red,
						Size: 3,
					},
				},
			},
		},
		{
			Name: "inserting past end paints empty gap",
			Paints: []paint{
				{0, 3, red, 2},
			},
			Result: &midterm.Canvas{
				Rows: []*midterm.Region{
					{
						F:    midterm.EmptyFormat,
						Size: 3,
						Next: &midterm.Region{
							F:    red,
							Size: 2,
						},
					},
				},
			},
		},
		{
			Name: "inserting in the middle of a gap",
			Paints: []paint{
				{0, 3, red, 2},
				{0, 1, green, 5},
			},
			Result: &midterm.Canvas{
				Rows: []*midterm.Region{
					{
						F:    midterm.EmptyFormat,
						Size: 1,
						Next: &midterm.Region{
							F:    green,
							Size: 5,
							Next: &midterm.Region{
								F:    midterm.EmptyFormat,
								Size: 2,
								Next: &midterm.Region{
									F:    red,
									Size: 2,
								},
							},
						},
					},
				},
			},
		},
		{
			Name: "inserting on boundary grows",
			Paints: []paint{
				{0, 0, midterm.EmptyFormat, 0},
				{0, 1, midterm.EmptyFormat, 0},
				{0, 1, midterm.EmptyFormat, 1},
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
			Name: "within region with same format grows region",
			Paints: []paint{
				{0, 0, red, 0},
				{0, 1, red, 0},
				{0, 2, red, 0},
				{0, 1, red, 1},
			},
			Result: &midterm.Canvas{
				Rows: []*midterm.Region{
					{
						F:    red,
						Size: 4,
						Next: nil,
					},
				},
			},
		},
		{
			Name: "within region with different format splits the region",
			Paints: []paint{
				{0, 0, red, 0},
				{0, 1, red, 0},
				{0, 2, red, 0},
				{0, 3, red, 0},
				{0, 1, green, 1},
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
								Size: 3,
							},
						},
					},
				},
			},
		},
		{
			Name: "different format between regions at start of next region",
			Paints: []paint{
				{0, 0, red, 0},
				{0, 1, red, 0},
				{0, 2, red, 0},
				{0, 3, green, 0},
				{0, 4, green, 0},
				{0, 5, green, 0},
				{0, 3, blue, 1},
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
								Size: 3,
							},
						},
					},
				},
			},
		},
		{
			Name: "same format between regions at start of next region",
			Paints: []paint{
				{0, 0, red, 0},
				{0, 1, red, 0},
				{0, 2, red, 0},
				{0, 3, green, 0},
				{0, 4, green, 0},
				{0, 5, green, 0},
				{0, 3, green, 1},
			},
			Result: &midterm.Canvas{
				Rows: []*midterm.Region{
					{
						F:    red,
						Size: 3,
						Next: &midterm.Region{
							F:    green,
							Size: 4,
						},
					},
				},
			},
		},
		{
			Name: "painting beyond the end of the region",
			Paints: []paint{
				{0, 0, red, 0},
				{0, 1, red, 0},
				{0, 2, red, 0},
				{0, 4, green, 1},
				{0, 5, green, 1},
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
			Name: "before a single width region",
			Paints: []paint{
				{0, 0, red, 0},
				{0, 1, red, 0},
				{0, 2, green, 0},
				{0, 3, blue, 0},
				{0, 2, brightGreen, 1},
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
								Next: &midterm.Region{
									F:    blue,
									Size: 1,
								},
							},
						},
					},
				},
			},
		},
		{
			Name: "before the start of a wider region",
			Paints: []paint{
				{0, 0, red, 0},
				{0, 1, red, 0},
				{0, 2, green, 0},
				{0, 3, green, 0},
				{0, 2, brightGreen, 1},
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
								Size: 2,
							},
						},
					},
				},
			},
		},
		{
			Name: "splitting the end of a wider region",
			Paints: []paint{
				{0, 0, red, 0},
				{0, 1, red, 0},
				{0, 2, green, 0},
				{0, 3, green, 0},
				{0, 4, green, 0},
				{0, 4, brightGreen, 1},
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
									F:    green,
									Size: 1,
								},
							},
						},
					},
				},
			},
		},
		{
			Name: "overwriting the end of a wider region with another region after it",
			Paints: []paint{
				{0, 0, red, 0},
				{0, 1, red, 0},
				{0, 2, green, 0},
				{0, 3, green, 0},
				{0, 4, green, 0},
				{0, 5, blue, 0},
				{0, 6, blue, 0},
				{0, 4, brightGreen, 1},
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
									F:    green,
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
		},
	} {
		t.Run(ex.Name, func(t *testing.T) {
			canvas := &midterm.Canvas{}
			for _, p := range ex.Paints {
				t.Logf("painting %+v", p)
				if p.insert == 0 {
					canvas.Paint(p.row, p.col, p.f)
				} else {
					canvas.Insert(p.row, p.col, p.f, p.insert)
				}
			}
			require.Equal(t, ex.Result, canvas)
		})
	}
}

func TestCanvasDelete(t *testing.T) {
	t.Skip("WIP")
	type paint struct {
		row, col int
		f        midterm.Format
		delete   int
	}

	type PaintExample struct {
		Name   string
		Paints []paint
		Result *midterm.Canvas
	}

	for _, ex := range []PaintExample{
		{
			Name: "initial paint followed by delete",
			Paints: []paint{
				{0, 0, midterm.EmptyFormat, 0},
				{0, 0, midterm.Format{}, 1},
			},
			Result: &midterm.Canvas{
				Rows: []*midterm.Region{
					{
						F:    midterm.EmptyFormat,
						Size: 0,
						Next: nil,
					},
				},
			},
		},
		{
			Name: "deleting more than is available",
			Paints: []paint{
				{0, 0, midterm.EmptyFormat, 0},
				{0, 0, midterm.EmptyFormat, 0},
				{0, 0, midterm.Format{}, 5},
			},
			Result: &midterm.Canvas{
				Rows: []*midterm.Region{
					{
						F:    midterm.EmptyFormat,
						Size: 0,
						Next: nil,
					},
				},
			},
		},
		{
			Name: "deleting from empty gap",
			Paints: []paint{
				{0, 3, red, 0},
				{0, 0, midterm.Format{}, 1},
			},
			Result: &midterm.Canvas{
				Rows: []*midterm.Region{
					{
						F:    midterm.EmptyFormat,
						Size: 2,
						Next: &midterm.Region{
							F:    red,
							Size: 1,
						},
					},
				},
			},
		},
		{
			Name: "deleting past end does nothing",
			Paints: []paint{
				{0, 0, midterm.EmptyFormat, 0},
				{0, 1, midterm.EmptyFormat, 0},
				{0, 2, midterm.EmptyFormat, 0},
				{0, 3, midterm.Format{}, 1},
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
			Name: "deleting within middle of region shrinks it",
			Paints: []paint{
				{0, 0, red, 0},
				{0, 1, red, 0},
				{0, 2, red, 0},
				{0, 1, red, 1},
			},
			Result: &midterm.Canvas{
				Rows: []*midterm.Region{
					{
						F:    red,
						Size: 2,
						Next: nil,
					},
				},
			},
		},
		{
			Name: "deleting within middle of region does not shrink beyond deletion point",
			Paints: []paint{
				{0, 0, red, 0},
				{0, 1, red, 0},
				{0, 2, red, 0},
				{0, 1, red, 3},
			},
			Result: &midterm.Canvas{
				Rows: []*midterm.Region{
					{
						F:    red,
						Size: 1,
						Next: nil,
					},
				},
			},
		},
		{
			Name: "different format between regions at start of next region",
			Paints: []paint{
				{0, 0, red, 0},
				{0, 1, red, 0},
				{0, 2, red, 0},
				{0, 3, green, 0},
				{0, 4, green, 0},
				{0, 5, green, 0},
				{0, 3, midterm.Format{}, 1},
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
			Paints: []paint{
				{0, 0, red, 0},
				{0, 1, red, 0},
				{0, 2, red, 0},
				{0, 3, green, 0},
				{0, 4, green, 0},
				{0, 5, green, 0},
				{0, 3, green, 0},
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
			Paints: []paint{
				{0, 0, red, 0},
				{0, 1, red, 0},
				{0, 2, red, 0},
				{0, 4, green, 0},
				{0, 5, green, 0},
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
			Paints: []paint{
				{0, 0, red, 0},
				{0, 1, red, 0},
				{0, 2, green, 0},
				{0, 3, blue, 0},
				{0, 2, brightGreen, 0},
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
			Paints: []paint{
				{0, 0, red, 0},
				{0, 1, red, 0},
				{0, 2, green, 0},
				{0, 3, green, 0},
				{0, 2, brightGreen, 0},
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
			Paints: []paint{
				{0, 0, red, 0},
				{0, 1, red, 0},
				{0, 2, green, 0},
				{0, 3, green, 0},
				{0, 4, green, 0},
				{0, 4, brightGreen, 0},
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
			Paints: []paint{
				{0, 0, red, 0},
				{0, 1, red, 0},
				{0, 2, green, 0},
				{0, 3, green, 0},
				{0, 4, green, 0},
				{0, 5, blue, 0},
				{0, 6, blue, 0},
				{0, 4, brightGreen, 0},
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
			Paints: []paint{
				{0, 0, red, 0},
				{0, 1, green, 0},
				{0, 0, blue, 0},
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
			Paints: []paint{
				{0, 0, red, 0},
				{0, 1, green, 0},
				{0, 1, blue, 0},
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
			Paints: []paint{
				{0, 0, red, 0},
				{0, 1, red, 0},
				{0, 2, green, 0},
				{0, 3, green, 0},
				{0, 4, green, 0},
				{0, 3, brightGreen, 0},
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
				if p.delete == 0 {
					canvas.Paint(p.row, p.col, p.f)
				} else {
					canvas.Delete(p.row, p.col, p.delete)
				}
			}
			require.Equal(t, ex.Result, canvas)
		})
	}
}
