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
uniform float aspect;

out vec3 VertPos;
out vec2 VertUV;
out vec3 VertColor;
out vec3 VertNormal;

/* But we deliver this..
{Name: "position", Type: glhf.Vec3},  -> vec3 position   //current point on line
{Name: "texCoord", Type: glhf.Vec2}, X-Coordinate  -> vec3 direction  //a sign, -1 or 1
{Name: "vertexColor", Type: glhf.Vec3}, -> vec3 previous   //previous point on line
{Name: "normal", Type: glhf.Vec3}, -> vec3 next       //next point on line
*/

// adapted from: https://github.com/mattdesl/webgl-lines/blob/master/projected/vert.glsl
void drawLine() {
    int miter = 0;// hard-coded for now, set to 1 for miter join

    float direction = texCoord.x;
    vec3 next = normal;
    vec3 previous = vertexColor;

    vec2 aspectVec = vec2(aspect, 1.0);
    mat4 projViewModel = camProjectionView * modelTransform;
    vec4 previousProjected = projViewModel * vec4(previous, 1.0);
    vec4 currentProjected = projViewModel * vec4(position, 1.0);
    vec4 nextProjected = projViewModel * vec4(next, 1.0);

    //get 2D screen space with W divide and aspect correction
    vec2 currentScreen = currentProjected.xy / currentProjected.w * aspectVec;
    vec2 previousScreen = previousProjected.xy / previousProjected.w * aspectVec;
    vec2 nextScreen = nextProjected.xy / nextProjected.w * aspectVec;

    float len = thickness;
    float orientation = direction;

    //starting point uses (next - current)
    vec2 dir = vec2(0.0);
    if (currentScreen == previousScreen) {
        dir = normalize(nextScreen - currentScreen);
    }
    //ending point uses (current - previous)
    else if (currentScreen == nextScreen) {
        dir = normalize(currentScreen - previousScreen);
    }
    //somewhere in middle, needs a join
    else {
        //get directions from (C - B) and (B - A)
        vec2 dirA = normalize((currentScreen - previousScreen));
        if (miter == 1) {
            vec2 dirB = normalize((nextScreen - currentScreen));
            //now compute the miter join normal and length
            vec2 tangent = normalize(dirA + dirB);
            vec2 perp = vec2(-dirA.y, dirA.x);
            vec2 miter = vec2(-tangent.y, tangent.x);
            dir = tangent;
            len = thickness / dot(miter, perp);
        } else {
            dir = dirA;
        }
    }
    vec2 normal = vec2(-dir.y, dir.x);
    normal *= len/2.0;
    normal.x /= aspect;

    vec4 offset = vec4(normal * orientation, 0.0, 1.0);
    gl_Position = currentProjected + offset;
    gl_PointSize = 1.0;
}


void main() {
    if (drawMode == 4) {
        drawLine();
        VertUV = vec2(0.0);
        VertColor = vec3(1.0);
        VertNormal = vec3(0.0);
    } else {
        gl_Position = camProjectionView * modelTransform * vec4(position, 1.0);
        VertUV = texCoord;
        VertColor = vertexColor;
        VertNormal = normal;
    }

    // pass-through for fragment shader
    VertPos = position;

}