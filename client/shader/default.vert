#version 330 core

in vec3 position;
in vec2 texCoord;
in vec3 vertexColor;
in vec3 normal;

uniform mat4 camProjectionView;
uniform mat4 modelTransform;

uniform int drawMode;
uniform vec4 color;

uniform float thickness;
uniform vec2 viewport;

uniform float multi;

out vec3 VertPos;
out vec2 VertUV;
out vec3 VertColor;
out vec3 VertNormal;

/* Expects
uniform vec2 viewport;
uniform mat4 model, view, projection;
uniform float antialias, thickness, linelength;
attribute vec3 prev, curr, next;
attribute vec2 uv;
varying vec2 v_uv;
*/
// still missing: linelength, uv
void drawLine() {
    mat4 projViewModel = camProjectionView * modelTransform;

    float antialias = 1.0;// hard-coded for now
    float linelength = multi;

    vec3 curr = position;
    vec3 next = normal;
    vec3 prev = vertexColor;

    // Normalized device coordinates
    vec4 NDC_prev = projViewModel * vec4(prev.xyz, 1.0);
    vec4 NDC_curr = projViewModel * vec4(curr.xyz, 1.0);
    vec4 NDC_next = projViewModel * vec4(next.xyz, 1.0);

    // Viewport (screen) coordinates
    vec2 screen_prev = viewport * ((NDC_prev.xy/NDC_prev.w) + 1.0)/2.0;
    vec2 screen_curr = viewport * ((NDC_curr.xy/NDC_curr.w) + 1.0)/2.0;
    vec2 screen_next = viewport * ((NDC_next.xy/NDC_next.w) + 1.0)/2.0;

    // Compute tickness according to line orientation (through surface normal)
    vec4 sNormal = modelTransform*vec4(curr.xyz, 1.0);
    VertNormal = sNormal.xyz;
    if (sNormal.z < 0.0)
    VertColor = vec3(thickness/2.0);
    else
    VertColor = vec3(thickness*(pow(sNormal.z, 0.5)+1.0)/2.0);

    vec2 twoDeePos;
    float w = thickness/2.0 + antialias;
    vec2 t0 = normalize(screen_curr.xy - screen_prev.xy);
    vec2 n0 = vec2(-t0.y, t0.x);
    vec2 t1 = normalize(screen_next.xy - screen_curr.xy);
    vec2 n1 = vec2(-t1.y, t1.x);
    VertUV = vec2(texCoord.x, texCoord.y*w);
    if (prev.xz == curr.xz) {
        VertUV.x = -w;
        twoDeePos = screen_curr.xy - w*t1 + texCoord.y*w*n1;
    } else if (curr.xz == next.xz) {
        VertUV.x = linelength+w;
        twoDeePos = screen_curr.xy + w*t0 + texCoord.y*w*n0;
    } else {
        vec2 miter = normalize(n0 + n1);
        // The max operator avoid glitches when miter is too large
        float dy = w / max(dot(miter, n1), 1.0);
        twoDeePos = screen_curr.xy + dy*texCoord.y*miter;
    }

    // Back to NDC coordinates
    gl_Position = vec4(2.0*twoDeePos/viewport-1.0, NDC_curr.z/NDC_curr.w, 1.0);
    /*
    vec2 twoDeePos;
    float w = thickness/2.0 + antialias;
    vec2 t0 = normalize(screen_curr.xy - screen_prev.xy);
    vec2 n0 = vec2(-t0.y, t0.x);
    vec2 t1 = normalize(screen_next.xy - screen_curr.xy);
    vec2 n1 = vec2(-t1.y, t1.x);
    VertUV = vec2(texCoord.x, texCoord.y*w);
    if (prev.xy == curr.xy) { // expects doubled vertices at the beginning and end
        VertUV.x = -w;
        twoDeePos = screen_curr.xy - w*t1 + texCoord.y*w*n1;
    } else if (curr.xy == next.xy) { // expects doubled vertices at the beginning and end
        VertUV.x = linelength+w;
        twoDeePos = screen_curr.xy + w*t0 + texCoord.y*w*n0;
    } else {
        vec2 miter = normalize(n0 + n1);
        // The max operator avoid glitches when miter is too large
        float dy = w / max(dot(miter, n1), 1.0);
        twoDeePos = screen_curr.xy + dy*texCoord.y*miter;
    }

    // Back to NDC coordinates
    gl_Position = vec4(2.0*twoDeePos/viewport-1.0, NDC_curr.z/NDC_curr.w, 1.0);
    */
}


/* But we deliver this..
{Name: "position", Type: glhf.Vec3},  -> vec3 position   //current point on line
{Name: "texCoord", Type: glhf.Vec2}, X-Coordinate  -> vec3 direction  //a sign, -1 or 1
{Name: "vertexColor", Type: glhf.Vec3}, -> vec3 previous   //previous point on line
{Name: "normal", Type: glhf.Vec3}, -> vec3 next       //next point on line
*/

void main() {
    if (drawMode == 4) {
        drawLine();// will set VertUV, VertNormal, VertColor.X = thickness
    } else {
        gl_Position = camProjectionView * modelTransform * vec4(position, 1.0);
        VertUV = texCoord;
        VertColor = vertexColor;
        VertNormal = normal;
    }

    // pass-through for fragment shader
    VertPos = position;

}