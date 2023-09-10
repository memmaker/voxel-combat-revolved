#version 330 core

in vec3 position;
in float lifetimeLeft;
in vec3 velocity;
in float sizeBegin;
//in vec3 origin;

uniform mat4 projection;
uniform mat4 modelView;

out VS_OUT {
    float lifetimeLeft;
    float sizeBegin;
} vs_out;

void main() {
    gl_Position = modelView * vec4(position, 1.0); // we just pass the position of this particle to the geometry shader
    vs_out.lifetimeLeft = lifetimeLeft; // from here on the pipeline looks good. setting this directly has visible resulta
    vs_out.sizeBegin = sizeBegin;
}
