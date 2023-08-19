#version 330 core

in vec2 VertPos;
in vec2 VertUV;// tl: 0,0 tr: 1,0 bl: 0,1 br: 1,1

uniform vec3 circleColor;
uniform float thickness;

out vec4 color;

void main() {
    float fade = thickness * 0.5;

    // to (-1..-1 -> 1..1)
    vec2 uvCentered = vec2((VertUV.x - 0.5) * 2.0, (VertUV.y - 0.5) * 2.0);
    float distance = 1.0 - length(uvCentered);// 0..2

    if (distance < 0.0) {
        discard;
    }
    color = vec4(smoothstep(0.0, fade, distance));
    color *= vec4(smoothstep(thickness, thickness - fade, distance));
    color *= vec4(circleColor, 1.0);
}

