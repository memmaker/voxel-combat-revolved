#version 330 core

layout (points) in;
// TODO: what is a meaningful max_vertices value?
layout (triangle_strip, max_vertices = 4) out;

uniform mat4 projection;
uniform mat4 modelView;
uniform vec3 camPos;
uniform vec3 camUp;

// take a point as input and output a quad as triangle strip
void main() {
    vec3 worldPosition = gl_in[0].gl_Position.xyz;

    vec4 inputPosition = gl_in[0].gl_Position;
    // clamp to 0.2
    //inputPosition.z = max(inputPosition.z, 0.2);
    //vec3 ndc = inputPosition.xyz / inputPosition.w;
    vec2 origin = inputPosition.xy;

    float quadSize = 0.7;

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