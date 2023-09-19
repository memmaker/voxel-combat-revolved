#version 330 core

layout (points) in;
layout (triangle_strip, max_vertices = 4) out;

in VS_OUT {
    vec2 size;
} gs_in[];

out GS_OUT {
    vec2 uv;
} gs_out;


uniform mat4 projection;

// take a point as input and output a quad as triangle strip
void main() {
    vec4 inputPosition = gl_in[0].gl_Position;

    vec2 origin = inputPosition.xy;

    float quadWidthHalf = gs_in[0].size.x * 0.5;
    float quadHeightHalf = gs_in[0].size.y * 0.5;

    vec2 up = vec2(0, 1);

    vec2 right = vec2(1, 0);

    vec2 topRight = origin + (up * quadHeightHalf) + (right * quadWidthHalf);
    gl_Position = projection * vec4(topRight, inputPosition.zw);
    gs_out.uv = vec2(1, 1);
    EmitVertex();

    vec2 topLeft = origin + (up * quadHeightHalf) - (right * quadWidthHalf);
    gl_Position = projection * vec4(topLeft, inputPosition.zw);
    gs_out.uv = vec2(0, 1);
    EmitVertex();

    vec2 bottomRight = origin - (up * quadHeightHalf) + (right * quadWidthHalf);
    gl_Position = projection * vec4(bottomRight, inputPosition.zw);
    gs_out.uv = vec2(1, 0);
    EmitVertex();

    vec2 bottomLeft = origin - (up * quadHeightHalf) - (right * quadWidthHalf);
    gl_Position = projection * vec4(bottomLeft, inputPosition.zw);
    gs_out.uv = vec2(0, 0);
    EmitVertex();

    EndPrimitive();
}