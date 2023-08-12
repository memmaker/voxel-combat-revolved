#version 330 core

in vec2 position;
in vec2 texCoord;

out vec2 VertPos;
out vec2 VertUV;
out vec3 VertColor;

uniform mat4 projection;
uniform mat4 camera;
uniform mat4 model;
uniform vec3 circleColor;
uniform float thickness;


void main() {
    gl_Position = projection * camera * model * vec4(position, 0.0, 1.0);

    // pass-through for fragment shader
    VertPos = position;
    VertUV = texCoord;
}