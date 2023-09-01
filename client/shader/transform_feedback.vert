#version 330 core

in vec3 inputPosition;
out vec3 outputPosition;


void main() {
    outputPosition = vec3(inputPosition.x + 0.01, inputPosition.y, inputPosition.z);
}
