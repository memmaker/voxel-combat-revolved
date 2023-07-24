#version 330 core

// Plan for the bit-packing
// 32 bits per vertex
// Bits 0..4: X position (0..31)
// xxxx xxxx xxxx xxxx xxxx xxxx xxxx xxxx
//                                  ^ ^^^^
//                                  X Position

// Bits 5..9: Y position (0..31)
// xxxx xxxx xxxx xxxx xxxx xxxx xxxx xxxx
//                            ^^ ^^^
//                            Y Position

// Bits 10..14: Z position (0..31)
// xxxx xxxx xxxx xxxx xxxx xxxx xxxx xxxx
//                      ^^^ ^^
//                      Z Position

// Bits 15..17: Normal (0..5)
// xxxx xxxx xxxx xxxx xxxx xxxx xxxx xxxx
//                  ^^ ^
//                  Normal Direction (1,0,0) (-1,0,0) (0,1,0) (0,-1,0) (0,0,1) (0,0,-1)

// Bits 18..31: Texture Index (0..4095)
// xxxx xxxx xxxx xxxx xxxx xxxx xxxx xxxx
// ^^^^ ^^^^ ^^^^ ^^
// Texture Index

//in vec3 position; // should be between 0..31
// 32 states needs 5 bits, so 5 bits per axis => 15 bits
//in vec3 normal;   // one of 6 directions, 0..5
// 6 states needs 3 bits, so 3 bits per normal
// what texture to apply? 1 byte?
// uv coords for texture? derive from texture index and position of vertex in quad (0..3)
// 4 states needs 2 bits, so 2 bits per uv coord

// that's 20 bits per vertex without the texture index
// if we assume a 32 bit int, that leaves 12 bits for the texture index
in int compressedValue;

flat out vec3 VertNormal;
invariant out vec3 VertPos;
flat out int VertTexIndex;
// set uniform locations

uniform mat4 projection;
uniform mat4 camera;
uniform mat4 model;
/*
type FaceType int32

const (
    yp FaceType = 0 // y positive
    yn FaceType = 1 // y negative

    xp FaceType = 2 // x positive
    xn FaceType = 3 // x negative
    zp FaceType = 4 // z positive
    zn FaceType = 5 // z negative
)
*/

vec3[6] normalLookup = vec3[6](
    vec3(1.0, 0.0, 0.0), // xp
    vec3(-1.0, 0.0, 0.0), // xn
    vec3(0.0, 1.0, 0.0), // yp
    vec3(0.0, -1.0, 0.0), // yn
    vec3(0.0, 0.0, 1.0), // zp
    vec3(0.0, 0.0, -1.0) // zn
);
/* Compression of position
xAxis := position.X         // 6 bits
yAxis := position.Y << 7   // 6 bits
zAxis := position.Z << 14   // 6 bits
*/
/* Compression of attributes
// 3 bits for the normal direction (0..5)
attributes := int32(normalDirection) << 21
// 8 bits for the texture index (0..255)
attributes |= int32(textureIndex) << 24
*/
void decompressVertex(int compressedValue, out ivec3 position, out int normalDir, out int textureIndex)
{
    position.x = compressedValue & 0x3F; // 0x3F = 0b111111
    position.y = (compressedValue >> 7) & 0x3F;
    position.z = (compressedValue >> 14) & 0x3F;
    normalDir = (compressedValue >> 21) & 0x7; // 0x7 = 0b111
    textureIndex = (compressedValue >> 24) & 0xFF; // 0xFF = 0b11111111
}


void main() {
    // decompress the vertex
    int normalDir;
    ivec3 position;

    decompressVertex(compressedValue, position, normalDir, VertTexIndex);

    // pass-through for fragment shader
    VertNormal = normalLookup[normalDir];
    VertPos = vec3(position.x, position.y, position.z);

    gl_Position = projection * camera * model * vec4(position, 1.0);
}
