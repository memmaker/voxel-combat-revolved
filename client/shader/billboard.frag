#version 330 core

out vec4 color;

in GS_OUT {
    vec2 uv;
} fs_in;

void main() {
    color = vec4(1.0, 1.0, 1.0, 1.0);
}
