#version 330 core
#extension all : disable

flat in vec3 VertNormal;
in vec3 VertPos;
in float VertWorldY;
in float VertLightLevel;
in float VertAO;
in float VertSky;
flat in uint VertTexIndex;

uniform sampler2D tex;

uniform vec3 global_light_direction;
uniform vec3 global_light_color;

uniform vec3 light_position;
uniform vec3 light_color;

uniform mat4 modelTransform;
// everything above this height is cut away, so players can look into buildings (< 0: off)
uniform float cutawayHeight;

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
    if (cutawayHeight >= 0.0 && VertWorldY > cutawayHeight + 0.001) {
        discard;
    }
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

    // light: sun or lamp level of the air in front of the face (0..15), smoothed across corners
    float light = VertLightLevel / 15.0;
    float direct = pow(light, 2.2);
    // faces turned towards the sun (+x, +y, +z, see sunDirection in lighting.go) catch more of it
    float shade = 0.62;                                  // undersides
    if (VertNormal.y > 0.5) shade = 1.0;                // tops
    else if (VertNormal.x > 0.5) shade = 0.88;          // east walls face the sun
    else if (VertNormal.z > 0.5) shade = 0.8;           // south walls
    else if (VertNormal.x < -0.5) shade = 0.66;         // west walls
    else if (VertNormal.z < -0.5) shade = 0.7;          // north walls
    shade = mix(0.78, shade, direct);                   // in the shade only the sky lights faces, evenly
    // the sky lights shaded spots: open squares stay readable, alleys and rooms get darker
    float ambient = 0.05 + 0.25 * VertSky;
    float brightness = shade * (ambient + (1.0 - ambient) * direct) * (0.9 + 0.1 * VertSky);
    vec3 tint = mix(vec3(0.72, 0.8, 1.0), vec3(1.06, 1.0, 0.92), direct); // cool sky shade, warm sunlight

    vec3 litColor = surfaceColor.rgb * brightness * tint * 1.15;
    vec3 toneMappedColor = toneMapping(litColor, 1.2, 1.0);
    // corners and wall bases get darker; applied after tone mapping, which would flatten it in sunlight
    toneMappedColor *= mix(0.4, 1.0, pow(VertAO, 0.8));
    color = vec4(toneMappedColor, surfaceColor.a);
}

