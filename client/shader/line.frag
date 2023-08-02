#version 330 core

out vec4 color;
uniform vec3 drawColor;
void main() {
    color = vec4(drawColor, 1.0);
}
