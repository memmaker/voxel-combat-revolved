#version 330 core

in vec3 position;
in vec3 normal;
in vec2 texCoord;

out vec3 VertNormal;
out vec3 VertPos;
out vec2 VertUV;

// set uniform locations

uniform mat4 projection;
uniform mat4 camera;
uniform mat4 model;

void main() {
    gl_Position = projection * camera * model * vec4(position, 1.0);

    // pass-through for fragment shader
    VertNormal = normal;
    VertPos = position;
    VertUV = texCoord;
}