#version 330 core

in vec3 position;
in vec2 size;

uniform mat4 modelView;

out VS_OUT {
    vec2 size;
} vs_out;

void main() {
    gl_Position = modelView * vec4(position, 1.0); // we just pass the position of this particle to the geometry shader
    vs_out.size = size;
}
