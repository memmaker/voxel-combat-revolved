//go:build js

package gl

// Constant values copied from go-gl/gl v4.1-core (only those the game uses).
const (
	ARRAY_BUFFER                                  = 0x8892
	ARRAY_BUFFER_BINDING                          = 0x8894
	BACK                                          = 0x0405
	BLEND                                         = 0x0BE2
	CLAMP_TO_BORDER                               = 0x812D
	COLOR_ATTACHMENT0                             = 0x8CE0
	COLOR_BUFFER_BIT                              = 0x00004000
	COMPILE_STATUS                                = 0x8B81
	CULL_FACE                                     = 0x0B44
	CURRENT_PROGRAM                               = 0x8B8D
	DEBUG_OUTPUT                                  = 0x92E0
	DEPTH_BUFFER_BIT                              = 0x00000100
	DEPTH_TEST                                    = 0x0B71
	DRAW_FRAMEBUFFER                              = 0x8CA9
	DRAW_FRAMEBUFFER_BINDING                      = 0x8CA6
	DST_ALPHA                                     = 0x0304
	ELEMENT_ARRAY_BUFFER                          = 0x8893
	ELEMENT_ARRAY_BUFFER_BINDING                  = 0x8895
	FALSE                                         = 0
	FILL                                          = 0x1B02
	FLOAT                                         = 0x1406
	FRAGMENT_SHADER                               = 0x8B30
	FRAMEBUFFER                                   = 0x8D40
	FRAMEBUFFER_BINDING                           = 0x8CA6
	FRONT_AND_BACK                                = 0x0408
	FUNC_ADD                                      = 0x8006
	GEOMETRY_SHADER                               = 0x8DD9
	INFO_LOG_LENGTH                               = 0x8B84
	INT                                           = 0x1404
	INTERLEAVED_ATTRIBS                           = 0x8C8C
	LESS                                          = 0x0201
	LINE                                          = 0x1B01
	LINEAR                                        = 0x2601
	LINES                                         = 0x0001
	LINK_STATUS                                   = 0x8B82
	MAX_TRANSFORM_FEEDBACK_INTERLEAVED_COMPONENTS = 0x8C8A
	NEAREST                                       = 0x2600
	NO_ERROR                                      = 0
	ONE                                           = 1
	ONE_MINUS_DST_ALPHA                           = 0x0305
	ONE_MINUS_SRC_ALPHA                           = 0x0303
	POINTS                                        = 0x0000
	RASTERIZER_DISCARD                            = 0x8C89
	READ_FRAMEBUFFER                              = 0x8CA8
	READ_FRAMEBUFFER_BINDING                      = 0x8CAA
	REPEAT                                        = 0x2901
	RGBA                                          = 0x1908
	SCISSOR_TEST                                  = 0x0C11
	SRC_ALPHA                                     = 0x0302
	STATIC_DRAW                                   = 0x88E4
	TEXTURE_2D                                    = 0x0DE1
	TEXTURE_BINDING_2D                            = 0x8069
	TEXTURE_BORDER_COLOR                          = 0x1004
	TEXTURE_MAG_FILTER                            = 0x2800
	TEXTURE_MIN_FILTER                            = 0x2801
	TEXTURE_WRAP_S                                = 0x2802
	TEXTURE_WRAP_T                                = 0x2803
	TRANSFORM_FEEDBACK                            = 0x8E22
	TRANSFORM_FEEDBACK_BUFFER                     = 0x8C8E
	TRIANGLES                                     = 0x0004
	TRIANGLE_STRIP                                = 0x0005
	UNSIGNED_BYTE                                 = 0x1401
	UNSIGNED_INT                                  = 0x1405
	VERSION                                       = 0x1F02
	VERTEX_ARRAY_BINDING                          = 0x85B5
	VERTEX_SHADER                                 = 0x8B31
	ZERO                                          = 0
)
