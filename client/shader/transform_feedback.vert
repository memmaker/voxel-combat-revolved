#version 330 core

uniform float deltaTime;
uniform float maxDistance;
uniform float lifetime;

in vec3 position;
in float lifetimeLeft;
in vec3 velocity;
in float sizeBegin;
in vec3 colorBegin;
in vec3 origin;

out VS_OUT {
    vec3 position;
    float lifetimeLeft;
    vec3 velocity;
    float sizeBegin;
    vec3 colorBegin;
    vec3 origin;
} vs_out;

void main() {
    vec3 newPos = position + (velocity * deltaTime);
    vec3 newVelocity = velocity;

    if (maxDistance > 0.0) {
        float distance = length(newPos - origin);
        if (distance > maxDistance) {
            newVelocity = -velocity;
            newPos = position + (newVelocity * deltaTime);
        }
    }

    float newLifetimeLeft = max(lifetimeLeft - deltaTime, 0.0);

    if (lifetime < -100.0) { // no reduction for infinite lifetime particles
        newLifetimeLeft = lifetimeLeft;
    } else if (lifetime < 0.0 && newLifetimeLeft <= 0) { // loop
        newLifetimeLeft = lifetime * -1.0;
        newPos = origin;
    }

    vs_out.position = newPos;
    vs_out.lifetimeLeft = newLifetimeLeft;

    vs_out.velocity = newVelocity;
    vs_out.sizeBegin = sizeBegin;

    vs_out.colorBegin = colorBegin;
    vs_out.origin = origin;
}
