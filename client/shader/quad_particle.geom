#version 330 core

layout (points) in;
layout (triangle_strip, max_vertices = 4) out;

in VS_OUT {
    float lifetimeLeft;
    float sizeBegin;
    vec4 origin;
} gs_in[];

out GS_OUT {
    float lifetimeLeft;
} gs_out;


uniform mat4 projection;
uniform mat4 modelView;
uniform float lifetime;
uniform float sizeEnd;


// take a point as input and output a quad as triangle strip
void main() {
    float lifeLeft = gs_in[0].lifetimeLeft;

    if (lifeLeft <= 0.0) {
        return;
    }
    float percentOfLifeLeft = lifeLeft / lifetime;
    float sizeBegin = gs_in[0].sizeBegin;

    float currentSize = mix(sizeBegin, sizeEnd, 1-percentOfLifeLeft);

    gs_out.lifetimeLeft = lifeLeft;

    vec4 inputPosition = gl_in[0].gl_Position;

    vec2 origin = inputPosition.xy;

    float quadSize = currentSize;

    vec2 up = vec2(0, 1);

    vec2 right = vec2(1, 0);

    vec2 va = origin + (up * quadSize) + (right * quadSize);
    gl_Position = projection * vec4(va, inputPosition.zw);
    EmitVertex();

    vec2 vb = origin + (up * quadSize) - (right * quadSize);
    gl_Position = projection * vec4(vb, inputPosition.zw);
    EmitVertex();

    vec2 vc = origin - (up * quadSize) + (right * quadSize);
    gl_Position = projection * vec4(vc, inputPosition.zw);
    EmitVertex();

    vec2 vd = origin - (up * quadSize) - (right * quadSize);
    gl_Position = projection * vec4(vd, inputPosition.zw);
    EmitVertex();

    EndPrimitive();
}