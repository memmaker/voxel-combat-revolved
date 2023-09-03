#version 410 core

uniform float deltaTime;

in vec3 position;
in float lifetimeLeft;
in vec3 velocity;
in float sizeBegin;

out VS_OUT {
    vec3 position;
    float lifetimeLeft;
    vec3 velocity;
    float sizeBegin;
} vs_out;

void main() {
    vs_out.position = position + (velocity * deltaTime);
    vs_out.lifetimeLeft = max(lifetimeLeft - deltaTime, 0.0);

    vs_out.velocity = velocity;
    vs_out.sizeBegin = sizeBegin;
}
