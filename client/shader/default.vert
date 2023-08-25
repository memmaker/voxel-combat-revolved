#version 330 core

in vec3 position;
in vec2 texCoord;
in vec3 vertexColor;
in vec3 normal;

uniform mat4 camProjectionView;
uniform mat4 modelTransform;

uniform int drawMode;
uniform vec4 color;

uniform float thickness;

out vec3 VertPos;
out vec2 VertUV;
out vec3 VertColor;
out vec3 VertNormal;


void main() {
    gl_Position = camProjectionView * modelTransform * vec4(position, 1.0);

    // pass-through for fragment shader
    VertPos = position;
    VertUV = texCoord;
    VertColor = vertexColor;
    VertNormal = normal;
}