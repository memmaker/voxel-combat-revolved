package util

import (
	bytes2 "bytes"
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/glhf"
	"github.com/qmuntal/gltf"
	"github.com/qmuntal/gltf/modeler"
	"image"
	"image/draw"
	_ "image/png"
	"io"
	"os"
	"path"
)

func LoadGLTFWithTextures(filename string) *CompoundMesh {
	result := LoadGLTF(filename)
	doc, err := gltf.Open(filename)
	if err != nil {
		println(err.Error())
		return nil
	}
	result.textures = tryLoadTextures(doc)
	return result
}

func LoadGLTF(filename string) *CompoundMesh {
	doc, err := gltf.Open(filename)
	if err != nil {
		println(err.Error())
		return nil
	}
	defaultSceneIndex := 0
	if doc.Scene != nil {
		defaultSceneIndex = int(*doc.Scene)
	}
	defaultScene := doc.Scenes[defaultSceneIndex]
	//fmt.Println(fmt.Sprintf("[LoadGLTF] Loading scene '%s' (1/%d)", defaultScene.name, len(doc.Scenes)))
	//fmt.Println(fmt.Sprintf("[LoadGLTF] Scene contains %d node(s)", len(defaultScene.Nodes)))
	result := &CompoundMesh{
		SamplerFrames:  make(map[string][][]float32),
		animationSpeed: 1.0,
	}

	flatMeshes := make([]*SimpleMesh, len(doc.Meshes))
	for meshIndex, _ := range doc.Meshes {
		flatMeshes[meshIndex] = loadMesh(doc, uint32(meshIndex))
	}
	flatNodes := make([]*MeshNode, len(doc.Nodes))
	for nodeIndex, node := range doc.Nodes {
		flatNodes[nodeIndex] = &MeshNode{
			name:         node.Name,
			animations:   make(map[string]*SimpleAnimationData),
			quatRotation: mgl32.QuatIdent(),
			translation:  mgl32.Vec3{0, 0, 0},
			scale:        mgl32.Vec3{1, 1, 1},
		}
		if node.Mesh != nil {
			flatNodes[nodeIndex].mesh = flatMeshes[*node.Mesh]
		}
	}
	for _, anim := range doc.Animations {
		//println(fmt.Sprintf("\nFound Animation: %s", anim.GameIdentifier))
		samplerFrames := make([][]float32, len(anim.Samplers))
		//samplerOutput := make([][][4]float32, len(anim.Samplers))
		for samplerIndex, sampler := range anim.Samplers {
			samplerFrames[samplerIndex] = inputKeyframesFromSampler(doc, sampler)
		}
		result.SamplerFrames[anim.Name] = samplerFrames
		for _, channel := range anim.Channels {
			var translationFrames [][3]float32
			var rotationFrames [][4]float32
			var scaleFrames [][3]float32
			nodeIndex := *channel.Target.Node
			node := flatNodes[nodeIndex]
			//println(fmt.Sprintf("\nNode for animation: %s", node.GameIdentifier))
			//println(fmt.Sprintf("Property provided by sampler: %s", channel.Targets.Path))
			samplerIndex := *channel.Sampler
			sampler := anim.Samplers[samplerIndex]
			outputValues := outputKeyframesFromSampler(doc, sampler)
			if _, exists := node.animations[anim.Name]; !exists {
				node.animations[anim.Name] = &SimpleAnimationData{}
			}

			switch channel.Target.Path { // THIS IS NEEDED FOR ANIMATION
			case gltf.TRSTranslation:
				translationFrames, _ = outputValues.([][3]float32)
				//println(fmt.Sprintf("Found # of Translation Keyframes: %d", len(translationFrames)))
				node.animations[anim.Name].TranslationFrames = translationFrames
				node.animations[anim.Name].TranslationSamplerIndex = samplerIndex
			case gltf.TRSRotation:
				rotationFrames, _ = outputValues.([][4]float32)
				//println(fmt.Sprintf("Found # of Rotation Keyframes: %d", len(rotationFrames)))
				node.animations[anim.Name].RotationFrames = rotationFrames
				node.animations[anim.Name].RotationSamplerIndex = samplerIndex
			case gltf.TRSScale:
				scaleFrames, _ = outputValues.([][3]float32)
				//println(fmt.Sprintf("Found # of Scale Keyframes: %d", len(scaleFrames)))
				node.animations[anim.Name].ScaleFrames = scaleFrames
				node.animations[anim.Name].ScaleSamplerIndex = samplerIndex
			}
		}
	}
	rootNode := buildNodeHierarchy(doc, flatNodes, defaultScene.Nodes[0], result.getSamplerFrames)
	result.RootNode = rootNode
	return result
}

func tryLoadTextures(doc *gltf.Document) []*glhf.Texture {
	results := make([]*glhf.Texture, len(doc.Textures))
	for texIndex, texture := range doc.Textures {
		//print(fmt.Sprintf("[LoadGLTFWithTextures] Texture at index %d ('%s'): ", texIndex, texture.name))
		imageSource := doc.Images[*texture.Source]
		if imageSource.IsEmbeddedResource() {
			embeddedTexture, err := loadEmbeddedTexture(imageSource)
			if err == nil {
				results[texIndex] = embeddedTexture
			}
		} else if imageSource.BufferView != nil {
			results[texIndex] = loadBufferTexture(doc, imageSource)
		} else {
			fileTexture, err := loadFileTexture(imageSource)
			if err == nil {
				results[texIndex] = fileTexture
			}
		}
	}

	return results
}

func loadFileTexture(imageSource *gltf.Image) (*glhf.Texture, error) {
	filename := imageSource.Name + ".png"
	filePath := path.Join("assets", "rawpng", filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		println(fmt.Sprintf("Error loading file texture: %s", err.Error()))
		return nil, err
	} else {
		// load bytes from file
		file, err := os.Open(filePath)
		if err != nil {
			println(fmt.Sprintf("Error loading file texture: %s", err.Error()))
		}
		loadedTexture, err := NewTextureFromReader(file, false)
		file.Close()
		//println(fmt.Sprintf("Loaded from file %s", filePath))
		return loadedTexture, nil
	}
}

func loadBufferTexture(doc *gltf.Document, imageSource *gltf.Image) *glhf.Texture {
	bufferView := doc.BufferViews[*imageSource.BufferView]
	buffer := doc.Buffers[bufferView.Buffer]
	data := buffer.Data[bufferView.ByteOffset : bufferView.ByteOffset+bufferView.ByteLength]
	loadedTexture, err := NewTextureFromReader(bytes2.NewReader(data), true)
	if err != nil {
		println(fmt.Sprintf("Error loading texture from buffer: %s", err.Error()))
	} else {
		//println(fmt.Sprintf("'%s' loaded from buffer", imageSource.name))
	}
	return loadedTexture
}

func loadEmbeddedTexture(image *gltf.Image) (*glhf.Texture, error) {
	data, err := image.MarshalData()
	if err != nil {
		println(fmt.Sprintf("Error loading embedded texture: %s", err.Error()))
		return nil, err
	}
	// save bytes as png
	loadedTexture, err := NewTextureFromReader(bytes2.NewReader(data), true)
	if err != nil {
		println(fmt.Sprintf("Error loading embedded texture: %s", err.Error()))
		return nil, err
	}
	if image.Name == "" {
		println("Loaded from embedded resource")
	} else {
		//println(fmt.Sprintf("'%s' loaded from embedded resource", image.name))
	}
	return loadedTexture, nil
}

// loadMesh loads all submeshes with their texture indices
func loadMesh(doc *gltf.Document, meshIndex uint32) *SimpleMesh {
	result := make([]*SubMesh, 0)
	mesh := doc.Meshes[meshIndex]
	for _, subMesh := range mesh.Primitives {
		currentSubmesh := &SubMesh{}
		material := doc.Materials[*subMesh.Material]
		if material.PBRMetallicRoughness.BaseColorTexture != nil {
			textureIndex := material.PBRMetallicRoughness.BaseColorTexture.Index
			currentSubmesh.TextureIndex = textureIndex
		}

		if subMesh.Mode != gltf.PrimitiveTriangles {
			println("WARNING: Only triangles are supported for now")
		}
		indexOfPositions := subMesh.Attributes["POSITION"]
		indexOfColors := subMesh.Attributes["COLOR_0"]
		indexOfNormals := subMesh.Attributes["NORMAL"]
		indexOfUVs := subMesh.Attributes["TEXCOORD_0"]

		positionAccessor := doc.Accessors[indexOfPositions]
		colorAccessor := doc.Accessors[indexOfColors]
		normalsAccessor := doc.Accessors[indexOfNormals]
		indicesAccessor := doc.Accessors[*subMesh.Indices]
		uvsAccessor := doc.Accessors[indexOfUVs]

		var vertBuffer [][3]float32
		var colorBuffer [][4]uint8
		var indicesBuffer []uint32
		var normalsBuffer [][3]float32
		var uvBuffer [][2]float32
		var err error
		vertBuffer, err = modeler.ReadPosition(doc, positionAccessor, vertBuffer)
		if err != nil {
			println(err)
			return nil
		}

	   colorBuffer, err = modeler.ReadColor(doc, colorAccessor, colorBuffer)
	   if err != nil {
		   println(err)
		   return nil
	   }

		indicesBuffer, err = modeler.ReadIndices(doc, indicesAccessor, indicesBuffer)
		if err != nil {
			println(err)
			return nil
		}
		normalsBuffer, err = modeler.ReadNormal(doc, normalsAccessor, normalsBuffer)
		if err != nil {
			println(err)
			return nil
		}
		uvBuffer, err = modeler.ReadTextureCoord(doc, uvsAccessor, uvBuffer)
		if err != nil {
			println(err)
			return nil
		}
		// merge them..
		var meshVertices []glhf.GlFloat
		for i := 0; i < len(vertBuffer); i++ {
			/*
			{Name: "position", Type: glhf.Vec3},
			{Name: "texCoord", Type: glhf.Vec2},
			{Name: "vertexColor", Type: glhf.Vec3},
			{Name: "normal", Type: glhf.Vec3},
			 */
			meshVertices = append(
				meshVertices,

				glhf.GlFloat(vertBuffer[i][0]),
				glhf.GlFloat(vertBuffer[i][1]),
				glhf.GlFloat(vertBuffer[i][2]),

				glhf.GlFloat(uvBuffer[i][0]),
				glhf.GlFloat(uvBuffer[i][1]),

				glhf.GlFloat(colorBuffer[i][0])/255.0,
				glhf.GlFloat(colorBuffer[i][1])/255.0,
				glhf.GlFloat(colorBuffer[i][2])/255.0,

				glhf.GlFloat(normalsBuffer[i][0]),
				glhf.GlFloat(normalsBuffer[i][1]),
				glhf.GlFloat(normalsBuffer[i][2]),
			)
		}
		currentSubmesh.Indices = indicesBuffer
		currentSubmesh.VertexData = meshVertices
		currentSubmesh.VertexCount = int(indicesAccessor.Count)
		result = append(result, currentSubmesh)
	}
	currentMesh := &SimpleMesh{
		// Vertex Format (translation, Normal, UV)
		// X0,Y0,Z0, NX0,NY0,NZ0, U0,V0
		SubMeshes: result,
	}
	return currentMesh
}

func buildNodeHierarchy(document *gltf.Document, meshNodes []*MeshNode, nodeIndex uint32, samplerFrameSource func(animationName string) [][]float32) *MeshNode {
	docNode := document.Nodes[nodeIndex]
	meshNode := meshNodes[nodeIndex]
	meshNode.SetSamplerSource(samplerFrameSource)

	meshNode.SetInitialTranslate(docNode.TranslationOrDefault())
	meshNode.SetInitialRotate(docNode.RotationOrDefault())
	meshNode.SetInitialScale(docNode.ScaleOrDefault())

	for _, childNodeIndex := range docNode.Children {
		meshNode.children = append(meshNode.children, buildNodeHierarchy(document, meshNodes, childNodeIndex, samplerFrameSource))
		childMeshNode := meshNodes[childNodeIndex]
		childMeshNode.parent = meshNode
	}
	return meshNode
}

func inputKeyframesFromSampler(doc *gltf.Document, sampler *gltf.AnimationSampler) []float32 {
	var result []float32
	inputAccessor := doc.Accessors[sampler.Input]
	inputAcr := &gltf.Accessor{
		BufferView: gltf.Index(*inputAccessor.BufferView), Count: inputAccessor.Count, Type: inputAccessor.Type, ComponentType: inputAccessor.ComponentType,
	}
	var inputBufferUntyped interface{}
	var err error
	inputBufferUntyped, err = modeler.ReadAccessor(doc, inputAcr, inputBufferUntyped)
	if err != nil {
		println(err)
		return nil
	}
	result = inputBufferUntyped.([]float32)
	return result
}

func outputKeyframesFromSampler(doc *gltf.Document, sampler *gltf.AnimationSampler) interface{} {
	outputAccessor := doc.Accessors[sampler.Output]
	outputAcr := &gltf.Accessor{
		BufferView: gltf.Index(*outputAccessor.BufferView), Count: outputAccessor.Count, Type: outputAccessor.Type, ComponentType: outputAccessor.ComponentType,
	}
	var inputBufferUntyped interface{}
	var err error
	inputBufferUntyped, err = modeler.ReadAccessor(doc, outputAcr, inputBufferUntyped)
	if err != nil {
		println(err)
		return nil
	}

	return inputBufferUntyped
}

// NewTextureFromReader creates a new texture from an io.Reader.
func NewTextureFromReader(r io.Reader, flipY bool) (*glhf.Texture, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	nrgba := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	if !flipY {
		draw.Draw(nrgba, nrgba.Bounds(), img, bounds.Min, draw.Src)
	} else {
		for y := 0; y < bounds.Dy(); y++ {
			for x := 0; x < bounds.Dx(); x++ {
				nrgba.Set(x, bounds.Dy()-y-1, img.At(x, y)) // flip the image on the CollisionY axis
			}
		}
	}

	var texture *glhf.Texture
	texture = glhf.NewTexture(
		nrgba.Bounds().Dx(),
		nrgba.Bounds().Dy(),
		false,
		nrgba.Pix,
	)
	return texture, nil
}
