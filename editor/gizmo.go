// Copyright 2017, Timothy Bogdala <tdb@animal-machine.com>
// See the LICENSE file for more details.

package editor

import (
	"fmt"

	mgl "github.com/go-gl/mathgl/mgl32"
	"github.com/tbogdala/fizzle"
	"github.com/tbogdala/fizzle/component"
	graphics "github.com/tbogdala/fizzle/graphicsprovider"
	"github.com/tbogdala/fizzle/scene"
	"github.com/tbogdala/glider"
)

const (
	axisDirX = 0
	axisDirY = 1
	axisDirZ = 2
)

// Gizmo is the transform gizmo that can be drawn in the editor.
type Gizmo struct {
	// Gizmo is the drawable gizmo object for the current operation.
	Gizmo *scene.VisibleEntity

	translate *fizzle.Renderable
	scale     *fizzle.Renderable
	rotate    *fizzle.Renderable

	// the last scale used while generating the gizmo
	lastScale float32

	mouseDown     bool // is the mouse considered to be down for event handling
	lastMouseX    float32
	lastMouseY    float32
	axisDir       int // the direction to apply the transform; use axisDirX || axisDirY || axisDirZ
	transformFunc func()
}

// CreateGizmo allocates a new gizmo and builds the renderable with the shader specified.
// (Shader should support Vert & VertColor)
func CreateGizmo(shader *fizzle.RenderShader) *Gizmo {
	g := new(Gizmo)

	// build the transform renderables
	g.buildRenderables(shader)

	// build the entity to render
	g.Gizmo = scene.NewVisibleEntity()
	g.Gizmo.Renderable = g.translate
	g.UpdateScale(1.0)

	return g
}

// generateColliders creates the colliders at the correct scaled location
// for the gizmo.
func (g *Gizmo) generateColliders(scale float32) {
	g.Gizmo.CoarseColliders = make([]glider.Collider, 0, 3)

	sphere := glider.NewSphere()
	sphere.Radius = 0.05 * scale
	sphere.Center = mgl.Vec3{0.9 * scale, 0.0, 0.0}
	g.Gizmo.CoarseColliders = append(g.Gizmo.CoarseColliders, sphere)

	sphere = glider.NewSphere()
	sphere.Radius = 0.05 * scale
	sphere.Center = mgl.Vec3{0.0, 0.9 * scale, 0.0}
	g.Gizmo.CoarseColliders = append(g.Gizmo.CoarseColliders, sphere)

	sphere = glider.NewSphere()
	sphere.Radius = 0.05 * scale
	sphere.Center = mgl.Vec3{0.0, 0.0, 0.9 * scale}
	g.Gizmo.CoarseColliders = append(g.Gizmo.CoarseColliders, sphere)

	for _, c := range g.Gizmo.CoarseColliders {
		sphere := c.(*glider.Sphere)
		fmt.Printf("Collider center: %v\n", sphere.Center)
	}

}

// UpdateScale modifies the the gizmo renderable for the current frame.
func (g *Gizmo) UpdateScale(scale float32) {
	// only update the gizmo scale if there's a significant change
	diff := g.lastScale - scale
	if diff < 0.0 {
		diff *= -1.0
	}
	if diff < 0.0001 {
		return
	}

	g.Gizmo.Renderable.Scale = mgl.Vec3{scale, scale, scale}
	g.generateColliders(scale)
	g.lastScale = scale
}

// OnLMBDown should be called by the owning component when the left mouse
// button is detected to be down. The Gizmo type will then take care of
// tracking state for the mouse positions. The coordinate [mx, my] should
// be normalized for screen size (divided by width and height).
func (g *Gizmo) OnLMBDown(mx, my float32, ray *glider.CollisionRay, active *component.Component) {
	if g.mouseDown == false {
		// if this is our first mouse down, reset the mouse position
		// tracking and test against axis handles
		g.mouseDown = true
		g.lastMouseX = -1.0
		g.lastMouseY = -1.0

		for axisNum, collider := range g.Gizmo.CoarseColliders {
			status, _ := collider.CollideVsRay(ray)
			if status != glider.NoIntersect {
				g.lastMouseX = mx
				g.lastMouseY = my
				g.axisDir = axisNum
				break
			}
		}

		// now that we initialized mouse position tracking and set the
		// transform direction, we can return from the handler and let
		// subsequent calls to this function deal with running
		// the transform function.
		return
	}

	// if mouse is down but the lastMouse positions are less than 0,
	// that means the mouse didn't hit an axis handle when it was first tracked,
	// so just don't do anything
	if g.lastMouseX < 0.0 || g.lastMouseY < 0.0 {
		return
	}

	// FIXME: for now, just use diffX to run the transform
	// FIXME: do more than translate
	diffX := g.lastScale * 10.0 * (g.lastMouseX - mx)
	var axisDir mgl.Vec3
	axisDir[g.axisDir] = 1.0
	diffDir := axisDir.Mul(diffX)
	active.Location = active.Location.Add(diffDir)

	// update the trackers before returning
	g.lastMouseX = mx
	g.lastMouseY = my
}

// OnLMBUp should be called by the owning component with the left mouse
// button is detected to be up.
func (g *Gizmo) OnLMBUp() {
	g.mouseDown = false
}

func addAxisToVBO(xmin, xmax, ymin, ymax, zmin, zmax, r, g, b, a float32, verts []float32, indexes []uint32, idxOffset uint32) ([]float32, []uint32, uint32) {
	/* Cube vertices are layed out like this:

	  +--------+           6          5
	/ |       /|
	+--------+ |        1          0        +Y
	| |      | |                            |___ +X
	| +------|-+           7          4    /
	|/       |/                           +Z
	+--------+          2          3

	*/
	verts = append(verts,
		xmax, ymax, zmax, r, g, b, a, xmin, ymax, zmax, r, g, b, a, xmin, ymin, zmax, r, g, b, a, xmax, ymin, zmax, r, g, b, a, // v0,v1,v2,v3 (front)
		xmax, ymax, zmin, r, g, b, a, xmax, ymax, zmax, r, g, b, a, xmax, ymin, zmax, r, g, b, a, xmax, ymin, zmin, r, g, b, a, // v5,v0,v3,v4 (right)
		xmax, ymax, zmin, r, g, b, a, xmin, ymax, zmin, r, g, b, a, xmin, ymax, zmax, r, g, b, a, xmax, ymax, zmax, r, g, b, a, // v5,v6,v1,v0 (top)
		xmin, ymax, zmax, r, g, b, a, xmin, ymax, zmin, r, g, b, a, xmin, ymin, zmin, r, g, b, a, xmin, ymin, zmax, r, g, b, a, // v1,v6,v7,v2 (left)
		xmax, ymin, zmax, r, g, b, a, xmin, ymin, zmax, r, g, b, a, xmin, ymin, zmin, r, g, b, a, xmax, ymin, zmin, r, g, b, a, // v3,v2,v7,v4 (bottom)
		xmin, ymax, zmin, r, g, b, a, xmax, ymax, zmin, r, g, b, a, xmax, ymin, zmin, r, g, b, a, xmin, ymin, zmin, r, g, b, a, // v6,v5,v4,v7 (back)
	)

	indexPattern := [...]uint32{
		0, 1, 2, 2, 3, 0,
		4, 5, 6, 6, 7, 4,
		8, 9, 10, 10, 11, 8,
		12, 13, 14, 14, 15, 12,
		16, 17, 18, 18, 19, 16,
		20, 21, 22, 22, 23, 20,
	}
	for _, idx := range indexPattern {
		indexes = append(indexes, idx+idxOffset)
	}

	return verts, indexes, idxOffset + 24
}

func buildAxisSet(a float32) (verts []float32, indexes []uint32, idxOffset uint32, facetotal uint32) {
	const axisCount = 3
	const min = float32(0.1)
	const max = float32(0.80)
	verts = make([]float32, 0, (4 * (3 + 4) * 6 * 3)) // 4 verts, 3+4 floats for pos&color, 6 faces, 3 total rectangles
	indexes = make([]uint32, 0, 24*3)
	verts, indexes, idxOffset = addAxisToVBO(min, max, -0.01, 0.01, -0.01, 0.01, 1.0, 0.0, 0.0, a, verts, indexes, idxOffset) // x-axis / red
	verts, indexes, idxOffset = addAxisToVBO(-0.01, 0.01, min, max, -0.01, 0.01, 0.0, 1.0, 0.0, a, verts, indexes, idxOffset) // y-axis / green
	verts, indexes, idxOffset = addAxisToVBO(-0.01, 0.01, -0.01, 0.01, min, max, 0.0, 0.0, 1.0, a, verts, indexes, idxOffset) // z-axis / blue

	return verts, indexes, idxOffset, 12 * axisCount
}

func assembleIntoRenderable(verts []float32, indexes []uint32, facecount uint32) *fizzle.Renderable {
	const floatSize = 4
	const uintSize = 4

	robj := fizzle.NewRenderable()
	robj.Material = fizzle.NewMaterial()
	robj.FaceCount = facecount
	robj.BoundingRect.Bottom = mgl.Vec3{-1, -1, -1}
	robj.BoundingRect.Top = mgl.Vec3{1, 1, 1}

	// create a VBO to hold the vertex data
	gfx := fizzle.GetGraphics()
	robj.Core.VertVBO = gfx.GenBuffer()
	robj.Core.VertColorVBO = robj.Core.VertVBO
	robj.Core.VertVBOOffset = 0
	robj.Core.VertColorVBOOffset = floatSize * 3
	robj.Core.VBOStride = floatSize * (3 + 4) // vert / vertcolor
	gfx.BindBuffer(graphics.ARRAY_BUFFER, robj.Core.VertVBO)
	gfx.BufferData(graphics.ARRAY_BUFFER, floatSize*len(verts), gfx.Ptr(&verts[0]), graphics.STATIC_DRAW)

	// create a VBO to hold the face indexes
	robj.Core.ElementsVBO = gfx.GenBuffer()
	gfx.BindBuffer(graphics.ELEMENT_ARRAY_BUFFER, robj.Core.ElementsVBO)
	gfx.BufferData(graphics.ELEMENT_ARRAY_BUFFER, uintSize*len(indexes), gfx.Ptr(&indexes[0]), graphics.STATIC_DRAW)

	return robj
}

func addTetrahedrons(verts []float32, indexes []uint32, idxOffset uint32, faceTotal uint32, a float32) ([]float32, []uint32, uint32, uint32) {
	newverts := append(verts,
		// +x
		0.850000, 0.035355, -0.035355, 1, 0, 0, a,
		0.950000, 0.000000, 0.000000, 1, 0, 0, a,
		0.850000, -0.035355, -0.035355, 1, 0, 0, a,
		0.850000, -0.035355, 0.035355, 1, 0, 0, a,
		0.850000, 0.035355, 0.035355, 1, 0, 0, a,

		// +z
		-0.035355, -0.035355, 0.850000, 0, 0, 1, a,
		0.000000, -0.000000, 0.950000, 0, 0, 1, a,
		0.035355, -0.035355, 0.850000, 0, 0, 1, a,
		0.035355, 0.035355, 0.850000, 0, 0, 1, a,
		-0.035355, 0.035355, 0.850000, 0, 0, 1, a,

		// +y
		-0.035355, 0.850000, -0.035355, 0, 1, 0, a,
		0.000000, 0.950000, 0.000000, 0, 1, 0, a,
		0.035355, 0.850000, -0.035355, 0, 1, 0, a,
		0.035355, 0.850000, 0.035355, 0, 1, 0, a,
		-0.035355, 0.850000, 0.035355, 0, 1, 0, a,
	)
	idxPattern := []uint32{
		0, 1, 2,
		2, 1, 3,
		3, 1, 4,
		4, 1, 0,
		2, 4, 0,
		2, 3, 4,

		5, 6, 7,
		7, 6, 8,
		8, 6, 9,
		9, 6, 5,
		7, 9, 5,
		7, 8, 9,

		10, 11, 12,
		12, 11, 13,
		13, 11, 14,
		14, 11, 10,
		12, 14, 10,
		12, 13, 14,
	}
	for _, i := range idxPattern {
		indexes = append(indexes, idxOffset+uint32(i))
	}
	return newverts, indexes, idxOffset + 15, faceTotal + 18
}

func addSquares(verts []float32, indexes []uint32, idxOffset uint32, faceTotal uint32, a float32) ([]float32, []uint32, uint32, uint32) {
	min := float32(0.85)
	max := float32(0.95)
	diff := float32(0.05)

	verts, indexes, idxOffset = addAxisToVBO(min, max, -diff, diff, -diff, diff, 1.0, 0.0, 0.0, a, verts, indexes, idxOffset) // x-axis / red
	verts, indexes, idxOffset = addAxisToVBO(-diff, diff, min, max, -diff, diff, 0.0, 1.0, 0.0, a, verts, indexes, idxOffset) // y-axis / green
	verts, indexes, idxOffset = addAxisToVBO(-diff, diff, -diff, diff, min, max, 0.0, 0.0, 1.0, a, verts, indexes, idxOffset) // z-axis / blue

	return verts, indexes, idxOffset, faceTotal + 24*3
}

func addToruses(verts []float32, indexes []uint32, idxOffset uint32, faceTotal uint32, a float32) ([]float32, []uint32, uint32, uint32) {
	verts = append(verts,
		0.9000, 0.0000, -0.0625, 1, 0, 0, a,
		0.9188, 0.0000, -0.0437, 1, 0, 0, a,
		0.8812, 0.0000, -0.0437, 1, 0, 0, a,
		0.9000, 0.0312, -0.0541, 1, 0, 0, a,
		0.9188, 0.0219, -0.0379, 1, 0, 0, a,
		0.8812, 0.0219, -0.0379, 1, 0, 0, a,
		0.9000, 0.0541, -0.0312, 1, 0, 0, a,
		0.9188, 0.0379, -0.0219, 1, 0, 0, a,
		0.8812, 0.0379, -0.0219, 1, 0, 0, a,
		0.9000, 0.0625, -0.0000, 1, 0, 0, a,
		0.9188, 0.0437, -0.0000, 1, 0, 0, a,
		0.8812, 0.0437, -0.0000, 1, 0, 0, a,
		0.9000, 0.0541, 0.0312, 1, 0, 0, a,
		0.9188, 0.0379, 0.0219, 1, 0, 0, a,
		0.8812, 0.0379, 0.0219, 1, 0, 0, a,
		0.9000, 0.0312, 0.0541, 1, 0, 0, a,
		0.9188, 0.0219, 0.0379, 1, 0, 0, a,
		0.8812, 0.0219, 0.0379, 1, 0, 0, a,
		0.9000, 0.0000, 0.0625, 1, 0, 0, a,
		0.9188, 0.0000, 0.0437, 1, 0, 0, a,
		0.8812, 0.0000, 0.0437, 1, 0, 0, a,

		-0.0625, 0.0000, 0.9000, 0, 0, 1, a,
		-0.0437, 0.0000, 0.9188, 0, 0, 1, a,
		-0.0437, 0.0000, 0.8812, 0, 0, 1, a,
		-0.0541, 0.0312, 0.9000, 0, 0, 1, a,
		-0.0379, 0.0219, 0.9188, 0, 0, 1, a,
		-0.0379, 0.0219, 0.8812, 0, 0, 1, a,
		-0.0312, 0.0541, 0.9000, 0, 0, 1, a,
		-0.0219, 0.0379, 0.9188, 0, 0, 1, a,
		-0.0219, 0.0379, 0.8812, 0, 0, 1, a,
		-0.0000, 0.0625, 0.9000, 0, 0, 1, a,
		-0.0000, 0.0437, 0.9188, 0, 0, 1, a,
		-0.0000, 0.0437, 0.8812, 0, 0, 1, a,
		0.0312, 0.0541, 0.9000, 0, 0, 1, a,
		0.0219, 0.0379, 0.9188, 0, 0, 1, a,
		0.0219, 0.0379, 0.8812, 0, 0, 1, a,
		0.0541, 0.0312, 0.9000, 0, 0, 1, a,
		0.0379, 0.0219, 0.9188, 0, 0, 1, a,
		0.0379, 0.0219, 0.8812, 0, 0, 1, a,
		0.0625, 0.0000, 0.9000, 0, 0, 1, a,
		0.0437, 0.0000, 0.9188, 0, 0, 1, a,
		0.0437, 0.0000, 0.8812, 0, 0, 1, a,

		-0.0000, 0.9000, 0.0625, 0, 1, 0, a,
		-0.0000, 0.9188, 0.0437, 0, 1, 0, a,
		-0.0000, 0.8812, 0.0437, 0, 1, 0, a,
		0.0312, 0.9000, 0.0541, 0, 1, 0, a,
		0.0219, 0.9188, 0.0379, 0, 1, 0, a,
		0.0219, 0.8812, 0.0379, 0, 1, 0, a,
		0.0541, 0.9000, 0.0312, 0, 1, 0, a,
		0.0379, 0.9188, 0.0219, 0, 1, 0, a,
		0.0379, 0.8812, 0.0219, 0, 1, 0, a,
		0.0625, 0.9000, 0.0000, 0, 1, 0, a,
		0.0437, 0.9188, 0.0000, 0, 1, 0, a,
		0.0437, 0.8812, 0.0000, 0, 1, 0, a,
		0.0541, 0.9000, -0.0312, 0, 1, 0, a,
		0.0379, 0.9188, -0.0219, 0, 1, 0, a,
		0.0379, 0.8812, -0.0219, 0, 1, 0, a,
		0.0312, 0.9000, -0.0541, 0, 1, 0, a,
		0.0219, 0.9188, -0.0379, 0, 1, 0, a,
		0.0219, 0.8812, -0.0379, 0, 1, 0, a,
		0.0000, 0.9000, -0.0625, 0, 1, 0, a,
		0.0000, 0.9188, -0.0437, 0, 1, 0, a,
		0.0000, 0.8812, -0.0437, 0, 1, 0, a,
	)

	idxPattern := []uint32{
		3, 1, 0,
		4, 2, 1,
		2, 3, 0,
		3, 7, 4,
		7, 5, 4,
		5, 6, 3,
		9, 7, 6,
		10, 8, 7,
		11, 6, 8,
		9, 13, 10,
		13, 11, 10,
		11, 12, 9,
		12, 16, 13,
		16, 14, 13,
		17, 12, 14,
		18, 16, 15,
		19, 17, 16,
		20, 15, 17,
		1, 2, 0,
		20, 19, 18,
		21, 25, 22,
		25, 23, 22,
		26, 21, 23,
		27, 25, 24,
		28, 26, 25,
		29, 24, 26,
		30, 28, 27,
		31, 29, 28,
		32, 27, 29,
		33, 31, 30,
		34, 32, 31,
		35, 30, 32,
		36, 34, 33,
		37, 35, 34,
		35, 36, 33,
		36, 40, 37,
		40, 38, 37,
		41, 36, 38,
		22, 23, 21,
		41, 40, 39,
		42, 46, 43,
		46, 44, 43,
		44, 45, 42,
		48, 46, 45,
		49, 47, 46,
		50, 45, 47,
		51, 49, 48,
		52, 50, 49,
		53, 48, 50,
		51, 55, 52,
		55, 53, 52,
		53, 54, 51,
		57, 55, 54,
		58, 56, 55,
		56, 57, 54,
		60, 58, 57,
		61, 59, 58,
		62, 57, 59,
		43, 44, 42,
		62, 61, 60,
		3, 4, 1,
		4, 5, 2,
		2, 5, 3,
		3, 6, 7,
		7, 8, 5,
		5, 8, 6,
		9, 10, 7,
		10, 11, 8,
		11, 9, 6,
		9, 12, 13,
		13, 14, 11,
		11, 14, 12,
		12, 15, 16,
		16, 17, 14,
		17, 15, 12,
		18, 19, 16,
		19, 20, 17,
		20, 18, 15,
		21, 24, 25,
		25, 26, 23,
		26, 24, 21,
		27, 28, 25,
		28, 29, 26,
		29, 27, 24,
		30, 31, 28,
		31, 32, 29,
		32, 30, 27,
		33, 34, 31,
		34, 35, 32,
		35, 33, 30,
		36, 37, 34,
		37, 38, 35,
		35, 38, 36,
		36, 39, 40,
		40, 41, 38,
		41, 39, 36,
		42, 45, 46,
		46, 47, 44,
		44, 47, 45,
		48, 49, 46,
		49, 50, 47,
		50, 48, 45,
		51, 52, 49,
		52, 53, 50,
		53, 51, 48,
		51, 54, 55,
		55, 56, 53,
		53, 56, 54,
		57, 58, 55,
		58, 59, 56,
		56, 59, 57,
		60, 61, 58,
		61, 62, 59,
		62, 60, 57,
	}

	for _, i := range idxPattern {
		indexes = append(indexes, idxOffset+uint32(i))
	}
	return verts, indexes, idxOffset + 63, faceTotal + uint32(len(idxPattern)/3)
}

func (g *Gizmo) buildRenderables(shader *fizzle.RenderShader) {
	const axisFaceCount = 12 * 3
	const alpha = 0.5

	// build the translate gizmo
	verts, indexes, idxOffset, faceTotal := buildAxisSet(alpha)
	verts, indexes, idxOffset, faceTotal = addTetrahedrons(verts, indexes, idxOffset, faceTotal, alpha)
	g.translate = assembleIntoRenderable(verts, indexes, faceTotal)
	g.translate.Material.Shader = shader

	// build the scale gizmo
	verts, indexes, idxOffset, faceTotal = buildAxisSet(alpha)
	verts, indexes, idxOffset, faceTotal = addSquares(verts, indexes, idxOffset, faceTotal, alpha)
	g.scale = assembleIntoRenderable(verts, indexes, faceTotal)
	g.scale.Material.Shader = shader

	// build the rotate gizmo
	verts, indexes, idxOffset, faceTotal = buildAxisSet(alpha)
	verts, indexes, idxOffset, faceTotal = addToruses(verts, indexes, idxOffset, faceTotal, alpha)
	g.rotate = assembleIntoRenderable(verts, indexes, faceTotal)
	g.rotate.Material.Shader = shader
}