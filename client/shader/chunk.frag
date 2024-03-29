#version 330 core
#extension all : disable

flat in vec3 VertNormal;
in vec3 VertPos;
in float VertLightLevel;
flat in uint VertTexIndex;

uniform sampler2D tex;

uniform vec3 global_light_direction;
uniform vec3 global_light_color;

uniform vec3 light_position;
uniform vec3 light_color;

uniform mat4 modelTransform;

out vec4 color;

vec3 toneMapping(vec3 hdrColor, float gamma, float exposure) {
    vec3 mapped = vec3(1.0) - exp(-hdrColor * exposure);
    // gamma correction
    mapped = pow(mapped, vec3(1.0 / gamma));
    return mapped;
}

float diffuseBrightnessFromGlobalLight(vec3 worldNormal) {
    //calculate final color of the pixel, based on:
    // 1. The angle of incidence: brightness

    //calculate the cosine of the angle of incidence
    float brightness = dot(worldNormal, global_light_direction) + 0.5;
    brightness = clamp(brightness, 0.0, 1.0);

    return brightness;
}

// simple diffuse lighting using a point light
// requires:
//  - vertex position in world space
//  - normal of the surface in world space
//  - position & color of the light source
float diffuseBrightnessFromPointLight(vec3 worldNormal) {
    //calculate final color of the pixel, based on:
    // 1. The angle of incidence: brightness

    vec3 worldPosition = vec3(modelTransform * vec4(VertPos, 1.0));

    //calculate the vector from this pixels surface to the light source
    vec3 surfaceToLight = light_position - worldPosition;

    //calculate the cosine of the angle of incidence
    float brightness = dot(worldNormal, surfaceToLight) / (length(surfaceToLight) * length(worldNormal));
    brightness = clamp(brightness, 0.0, 1.0);

    return brightness;
}
void main() {
    //calculate normal in world coordinates
    //mat3 normalMatrix = transpose(inverse(mat3(modelTransform)));
    //vec3 worldNormal = normalize(normalMatrix * VertNormal);

    //calculate the location of this fragment (pixel) in world coordinates
    //float directional_brightness = diffuseBrightnessFromGlobalLight(worldNormal);
    //float point_brightness = diffuseBrightnessFromPointLight(worldNormal);
    //float brightness = clamp(directional_brightness + point_brightness, 0.0, 1.0);

    //vec4 surfaceColor = vec4(1, 0.2, 0.2, 1);
    // our uv coords are in steps of 1/16
    // so we are interpolating in between that interval
    float xOffset = float(VertTexIndex % uint(16));// tiles per row
    float yOffset = float(VertTexIndex / uint(16));// tiles per column

    float u = 0.0;
    float v = 0.0;

    if (VertNormal.z > 0.0) {
        u = (xOffset + fract(VertPos.x)) / 16.0;
        v = (yOffset + (1.0 - fract(VertPos.y))) / 16.0;
    } else if (VertNormal.z < 0.0) {
        u = (xOffset + (1.0 - fract(VertPos.x))) / 16.0;
        v = (yOffset + (1.0 - fract(VertPos.y))) / 16.0;
    } else if (VertNormal.x > 0.0) {
        u = (xOffset + (1.0 - fract(VertPos.z))) / 16.0;
        v = (yOffset + (1.0 - fract(VertPos.y))) / 16.0;
    } else if (VertNormal.x < 0.0) {
        u = (xOffset + fract(VertPos.z)) / 16.0;
        v = (yOffset + (1.0 - fract(VertPos.y))) / 16.0;
    } else if (VertNormal.y > 0.0) {
        u = (xOffset + fract(VertPos.x)) / 16.0;
        v = (yOffset + fract(VertPos.z)) / 16.0;//old and working
    } else if (VertNormal.y < 0.0) {
        u = (xOffset + (1.0 - fract(VertPos.x))) / 16.0;
        v = (yOffset + fract(VertPos.z)) / 16.0;//old and working
    }

    // clamp to 0-1 range
    u = clamp(u, 0.0, 1.0);
    v = clamp(v, 0.0, 1.0);

    vec4 surfaceColor = texture(tex, vec2(u, v));
    //vec4 surfaceColor = vec4(VertColor, 1.0);

    vec3 floodLight = vec3(2.0, 2.0, 2.0) * (VertLightLevel/15.0);

    vec3 litColor = floodLight * surfaceColor.rgb;
    vec3 toneMappedColor = toneMapping(litColor, 1.2, 1.0);
    color = vec4(toneMappedColor, surfaceColor.a);
}

