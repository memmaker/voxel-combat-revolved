package voxel

import (
	"compress/gzip"
	"encoding/binary"
	"github.com/Tnze/go-mc/nbt"
	"os"
)

/*
	TAG_Compound({
	    "entities": TAG_List([
	        TAG_Compound({
	            "namespace": TAG_String(),
	            "base_name": TAG_String(),
	            "x": TAG_Double(),
	            "y": TAG_Double(),
	            "z": TAG_Double(),
	            "nbt": TAG_Compound()
	        })
	        ...
	    ]),
	    "block_entities": TAG_List([
	        TAG_Compound({
	            "namespace": TAG_String(),
	            "base_name": TAG_String(),
	            "x": TAG_Int(),
	            "y": TAG_Int(),
	            "z": TAG_Int(),
	            "nbt": TAG_Compound()
	        })
	        ...
	    ]),
	    "blocks_array_type": TAG_Byte(),
	    "blocks": <See below>
	})
*/
type SectionBlockInfo struct {
	//Entities        []BlockEntity `nbt:"entities"`
	BlocksArrayType byte `nbt:"blocks_array_type"`
}
type ByteSection struct {
	//Entities        []BlockEntity `nbt:"entities"`
	BlockEntities []BlockEntity `nbt:"block_entities"`
	Blocks        []byte        `nbt:"blocks"`
}
type IntSection struct {
	//Entities        []BlockEntity `nbt:"entities"`
	BlockEntities []BlockEntity `nbt:"block_entities"`
	Blocks        []int         `nbt:"blocks"`
}

type BlockEntity struct {
	Namespace string `nbt:"namespace"`
	Name      string `nbt:"base_name"`
	X         int32  `nbt:"x"`
	Y         int32  `nbt:"y"`
	Z         int32  `nbt:"z"`
}

type AmuletMetadata struct {
	SelectionBoxes    []int32 `nbt:"selection_boxes"`
	SectionIndexTable []byte  `nbt:"section_index_table"`
	SectionVersion    byte    `nbt:"section_version"`
	ExportVersion     struct {
		Edition string  `nbt:"edition"`
		Version []int32 `nbt:"version"`
	} `nbt:"export_version"`
	BlockPalette []*BlockDefinition `nbt:"block_palette"`
	CreatedWith  string             `nbt:"created_with"`
}
type BlockDefinition struct {
	Name       string         `nbt:"blockname"`
	NameSpace  string         `nbt:"namespace"`
	Properties map[string]any `nbt:"properties"`
}
type Construction struct {
	Sections []*ConstructionSection
}

type ConstructionSection struct {
	Blocks        []*BlockDefinition
	ShapeX        uint8
	ShapeY        uint8
	ShapeZ        uint8
	MinBlockX     int32
	MinBlockY     int32
	MinBlockZ     int32
	BlockEntities []BlockEntity
}

func LoadConstruction(filename string) *Construction {
	fileReader, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	var magicNumber [8]byte
	err = binary.Read(fileReader, binary.BigEndian, &magicNumber)
	if err != nil {
		panic(err)
	}
	// to string
	magicNumberString := string(magicNumber[:])
	magic_num := "constrct"
	if magicNumberString != magic_num {
		panic("invalid magic number")
	}
	offset := len(magic_num)

	// 8 bytes + 32bit int = 12 bytes
	fileReader.Seek(int64(-offset), 2)

	var magicNumber2 [8]byte
	err = binary.Read(fileReader, binary.BigEndian, &magicNumber2)
	if err != nil {
		panic(err)
	}

	magicNumberString2 := string(magicNumber2[:])
	if magicNumberString2 != magic_num {
		panic("invalid magic number")
	}

	fileReader.Seek(int64(-offset-4), 2)
	// read one 32bit int
	var metaDataOffset int32
	err = binary.Read(fileReader, binary.BigEndian, &metaDataOffset)
	if err != nil {
		panic(err)
	}
	// seek to the start of the metadata
	fileReader.Seek(int64(metaDataOffset), 0)
	gzipReader, err := gzip.NewReader(fileReader)
	decoder := nbt.NewDecoder(gzipReader)
	var value AmuletMetadata
	_, decErr := decoder.Decode(&value)
	if decErr != nil {
		panic(decErr)
	}
	sectionTable := decodeSectionTable(value.SectionIndexTable)
	sections := make([]*ConstructionSection, len(sectionTable))
	for sIndex, section := range sectionTable {

		fileReader.Seek(int64(section.Offset), 0)
		// read gzipped section data
		gzipReader, err = gzip.NewReader(fileReader)
		if err != nil {
			panic(err)
		}
		sectionDecoder := nbt.NewDecoder(gzipReader)
		var sectionBlockType SectionBlockInfo
		var blockEntities []BlockEntity
		var blocks []*BlockDefinition
		_, decErr = sectionDecoder.Decode(&sectionBlockType)
		if decErr != nil {
			panic(decErr)
		}
		switch sectionBlockType.BlocksArrayType {
		case 7:
			var decodedSection ByteSection
			fileReader.Seek(int64(section.Offset), 0)
			gzipReader.Reset(fileReader)
			sectionDecoder = nbt.NewDecoder(gzipReader)
			_, decErr = sectionDecoder.Decode(&decodedSection)
			if decErr != nil {
				panic(decErr)
			}
			blockEntities = decodedSection.BlockEntities
			blocks = decodeBlocks(decodedSection.Blocks, section.ShapeX, section.ShapeY, section.ShapeZ, value.BlockPalette)
		case 11:
			var decodedSection IntSection
			fileReader.Seek(int64(section.Offset), 0)
			gzipReader.Reset(fileReader)
			sectionDecoder = nbt.NewDecoder(gzipReader)
			_, decErr = sectionDecoder.Decode(&decodedSection)
			if decErr != nil {
				panic(decErr)
			}
			blockEntities = decodedSection.BlockEntities
			blocks = decodeBlocks(decodedSection.Blocks, section.ShapeX, section.ShapeY, section.ShapeZ, value.BlockPalette)
		}
		cSection := &ConstructionSection{
			Blocks:        blocks,
			BlockEntities: blockEntities,
			ShapeX:        section.ShapeX,
			ShapeY:        section.ShapeY,
			ShapeZ:        section.ShapeZ,
			MinBlockX:     section.MinBlockX,
			MinBlockY:     section.MinBlockY,
			MinBlockZ:     section.MinBlockZ,
		}
		sections[sIndex] = cSection
	}

	return &Construction{Sections: sections}
}

func decodeBlocks[T int | byte](blocks []T, shapeX uint8, shapeY uint8, shapeZ uint8, palette []*BlockDefinition) []*BlockDefinition {
	//totalBlocks := int(shapeX) * int(shapeY) * int(shapeZ)
	/*
		if len(blocks) != totalBlocks {
			panic("not a byte array, convert to int16 or int32")
		}

	*/
	result := make([]*BlockDefinition, len(blocks))
	for i := 0; i < len(blocks)-1; i++ {
		block := blocks[i]
		blockDefinition := palette[block]
		result[i] = blockDefinition
	}
	return result
}

/*
The section_index_table is an Mx23 TAG_Byte_Array where M is the number of section data entries present in the construction file. May be empty if there are no section data entries.

The real format of the section_index_table is IIIBBBII where I is a uint32 and B is a uint8.

Each represents the following

III: The X, Y, and Z block coordinates of the minimum point of the section
BBB: The shape of the section in blocks in X, Y, Z order
I: The starting byte of the section data entry in the file
I: The byte length of the section data entry
*/

type SectionIndex struct {
	MinBlockX int32  // 4 bytes
	MinBlockY int32  // 4 bytes
	MinBlockZ int32  // 4 bytes => 12 bytes
	ShapeX    uint8  // 1 byte
	ShapeY    uint8  // 1 byte
	ShapeZ    uint8  // 1 byte => 3 bytes
	Offset    uint32 // 4 bytes
	Size      uint32 // 4 bytes => 8 bytes
	// 23 bytes per section
}

func decodeSectionTable(table []byte) []SectionIndex {
	// 23 bytes per section
	sectionCount := len(table) / 23
	sections := make([]SectionIndex, sectionCount)
	for i := 0; i < sectionCount; i++ {
		sections[i].MinBlockX = int32(binary.LittleEndian.Uint32(table[i*23 : i*23+4]))
		sections[i].MinBlockY = int32(binary.LittleEndian.Uint32(table[i*23+4 : i*23+8]))
		sections[i].MinBlockZ = int32(binary.LittleEndian.Uint32(table[i*23+8 : i*23+12]))
		sections[i].ShapeX = table[i*23+12]
		sections[i].ShapeY = table[i*23+13]
		sections[i].ShapeZ = table[i*23+14]
		sections[i].Offset = binary.LittleEndian.Uint32(table[i*23+15 : i*23+19])
		sections[i].Size = binary.LittleEndian.Uint32(table[i*23+19 : i*23+23])
	}
	return sections
}
