#version 330 core

in vec3 VertPos;
in vec2 VertUV;// tl: 0,0 tr: 1,0 bl: 0,1 br: 1,1
in vec3 VertColor;
in vec3 VertNormal;

uniform mat4 camProjectionView;
uniform mat4 modelTransform;

uniform int drawMode;
uniform vec3 color;

uniform float thickness;

uniform vec3 global_light_direction;
uniform vec3 global_light_color;

uniform vec3 light_position;
uniform vec3 light_color;

uniform sampler2D tex;

out vec4 fragmentColor;

float diffuseBrightnessFromGlobalLight(vec3 worldNormal) {
    //calculate final color of the pixel, based on:
    // 1. The angle of incidence: brightness

    //calculate the cosine of the angle of incidence
    float brightness = dot(worldNormal, global_light_direction) + 0.5;
    brightness = clamp(brightness, 0, 1);

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

    vec3 worldPosition = vec3(modelTransform * vec4(VertPos, 1));

    //calculate the vector from this pixels surface to the light source
    vec3 surfaceToLight = light_position - worldPosition;

    //calculate the cosine of the angle of incidence
    float brightness = dot(worldNormal, surfaceToLight) / (length(surfaceToLight) * length(worldNormal));
    brightness = clamp(brightness, 0, 1);

    return brightness;
}

void drawModel() {
    //calculate normal in world coordinates
    mat3 normalMatrix = transpose(inverse(mat3(modelTransform)));
    vec3 worldNormal = normalize(normalMatrix * VertNormal);

    //calculate the location of this fragment (pixel) in world coordinates
    float directional_brightness = diffuseBrightnessFromGlobalLight(worldNormal);
    float point_brightness = diffuseBrightnessFromPointLight(worldNormal);
    float brightness = clamp(directional_brightness + point_brightness, 0, 1);

    //vec4 surfaceColor = vec4(1, 0.2, 0.2, 1);
    vec4 surfaceColor = texture(tex, vec2(VertUV.x, 1-VertUV.y));// 1-Tex.y because texture is flipped
    if (surfaceColor.a == 0) {
        discard;
    }
    //vec4 surfaceColor = vec4(VertColor, 1.0);
    fragmentColor = vec4(brightness * light_color * surfaceColor.rgb, surfaceColor.a);
}

void drawCircle() {
    float fade = thickness * 0.5;

    // to (-1..-1 -> 1..1)
    vec2 uvCentered = vec2((VertUV.x - 0.5) * 2.0, (VertUV.y - 0.5) * 2.0);
    float distance = 1.0 - length(uvCentered);// 0..2

    if (distance < 0.0) {
        discard;
    }
    fragmentColor = vec4(smoothstep(0.0, fade, distance));
    fragmentColor *= vec4(smoothstep(thickness, thickness - fade, distance));
    //fragmentColor *= vec4(VertColor, 1.0);
    fragmentColor *= vec4(color, 1.0);

}


void main() {
    if (drawMode == 0) {
        drawModel();
    } else if (drawMode == 3) {
        drawCircle();
    }
}