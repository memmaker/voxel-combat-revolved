#version 330 core

out vec4 color;

in GS_OUT {
    float lifetimeLeft;
} fs_in;

uniform float lifetime;
uniform vec4 colorBegin;
uniform vec4 colorEnd;

void main() {
    float percentOfLifeLeft = fs_in.lifetimeLeft / lifetime;
    vec4 lerpedColor = mix(colorBegin, colorEnd, 1-percentOfLifeLeft);

    color = lerpedColor;
}
