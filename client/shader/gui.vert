#version 330 core

in vec2 position;
in vec2 texCoord;

out vec2 VertUV;

uniform mat4 projection;
uniform mat4 model;

void main() {
    gl_Position = projection * model * vec4(position, 0.0, 1.0);

    // pass-through for fragment shader
    VertUV = texCoord;
}