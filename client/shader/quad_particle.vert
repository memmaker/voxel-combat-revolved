#version 330 core

in vec3 inputPosition; // this is the state we received from the transform feedback
uniform mat4 projection;
uniform mat4 modelView;

void main() {
    gl_Position = modelView * vec4(inputPosition, 1.0); // we just pass the position of this particle to the geometry shader
}
