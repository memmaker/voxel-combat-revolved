#version 330 core

in vec3 VertPos;
in vec2 VertUV;// tl: 0,0 tr: 1,0 bl: 0,1 br: 1,1
in vec3 VertColor;
in vec3 VertNormal;

uniform mat4 camProjectionView;
uniform mat4 modelTransform;

uniform int drawMode;
uniform vec4 color;

uniform float thickness;

uniform vec3 global_light_direction;
uniform vec3 global_light_color;

uniform vec3 light_position;
uniform vec3 light_color;

uniform float multi;

uniform sampler2D tex;

out vec4 fragmentColor;

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

void drawTexturedQuads() {
    //calculate normal in world coordinates
    mat3 normalMatrix = transpose(inverse(mat3(modelTransform)));
    vec3 worldNormal = normalize(normalMatrix * VertNormal);

    //calculate the location of this fragment (pixel) in world coordinates
    float directional_brightness = diffuseBrightnessFromGlobalLight(worldNormal);
    float point_brightness = diffuseBrightnessFromPointLight(worldNormal);
    float brightness = clamp(directional_brightness + point_brightness, 0.0, 1.0);

    vec4 surfaceColor = texture(tex, vec2(VertUV.x, 1-VertUV.y));// 1-Tex.y because texture is flipped
    if (surfaceColor.a == 0) {
        discard;
    }
    fragmentColor = vec4(brightness * light_color * surfaceColor.rgb, surfaceColor.a);
}

void drawColoredQuads() {
    vec4 surfaceColor = vec4(VertColor, 1.0);// we probably wanna set the transparency here
    surfaceColor *= color;
    fragmentColor = surfaceColor;//vec4(brightness * light_color * surfaceColor.rgb, surfaceColor.a);
}

void drawColoredFadingQuads() {
    vec4 surfaceColor = vec4(VertColor, VertUV.y + 0.2);// we probably wanna set the transparency here
    surfaceColor *= color;
    fragmentColor = surfaceColor;//vec4(brightness * light_color * surfaceColor.rgb, surfaceColor.a);
}

void drawCircle() {
    float fade = thickness * 0.5;

    // to (-1..-1 -> 1..1)
    vec2 uvCentered = vec2((VertUV.x - 0.5) * 2.0, (VertUV.y - 0.5) * 2.0);
    float distance = 1.0 - length(uvCentered);// 0..2
    /*
    if (distance < 0.0) {
        discard;
    }
    */
    fragmentColor = vec4(smoothstep(0.0, fade, distance));
    fragmentColor *= vec4(smoothstep(thickness, thickness - fade, distance));
    //fragmentColor *= vec4(VertColor, 1.0);
    fragmentColor *= color;

}
// still missing: linelength, uv
void drawLine() {
    float antialias = 1.0;// hard-coded for now
    float linelength = multi;
    float adjustedThickness = VertColor.x;

    float d = 0.0;
    float w = adjustedThickness/2.0 - antialias;

    vec3 lineColor = color.rgb;

    if (VertNormal.z < 0.0)
    lineColor *= 0.75*vec3(pow(abs(VertNormal.z), .5));//*vec3(0.95, 0.75, 0.75);

    // Cap at start
    if (VertUV.x < 0.0)
    d = length(VertUV) - w;
    // Cap at end
    else if (VertUV.x >= linelength)
    d = length(VertUV - vec2(linelength, 0.0)) - w;
    // Body
    else
    d = abs(VertUV.y) - w;
    if (d < 0.0) {
        fragmentColor = vec4(lineColor, color.a);
    } else {
        d /= antialias;
        fragmentColor = vec4(lineColor.rgb, exp(-d*d));
    }
}


void main() {
    if (drawMode == 0) {
        drawTexturedQuads();
    } else if (drawMode == 1) {
        drawColoredQuads();
    } else if (drawMode == 2) {
        drawColoredFadingQuads();
    } else if (drawMode == 3) {
        drawCircle();
    } else if (drawMode == 4) {
        drawLine();
    }
}