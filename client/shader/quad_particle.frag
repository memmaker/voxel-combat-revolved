#version 330 core

out vec4 color;

in GS_OUT {
    float lifetimeLeft;
    vec3 color;
} fs_in;

uniform float lifetime;

void main() {
    float percentOfLifeLeft = fs_in.lifetimeLeft / lifetime;
    //vec4 lerpedColor = mix(colorBegin, colorEnd, 1-percentOfLifeLeft);
    color = vec4(fs_in.color, 1.0);
}
