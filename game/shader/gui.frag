#version 330 core

in vec3 VertPos;
in vec2 VertUV;

uniform sampler2D tex;

out vec4 color;

void main() {
    color = texture(tex, VertUV);
}

