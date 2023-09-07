#version 330 core

in vec2 VertUV;

uniform sampler2D tex;
uniform vec4 appliedTintColor;
uniform vec4 discardedColor;

out vec4 color;

void main() {
 	vec4 texColor = texture(tex, VertUV);

 	if (texColor.rgb == discardedColor.rgb && discardedColor.a > 0.0) {
 		discard;
 	}

 	if (appliedTintColor.a > 0.0) {
 		texColor.rgb = texColor.rgb * appliedTintColor.rgb;
	}

    color = texColor;
}

