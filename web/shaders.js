export const meshVertex = `#version 300 es
precision highp float;
in vec3 aPosition;
in vec3 aNormal;
in vec3 aColor;
in float aConfidence;
in float aDeviation;
in float aFeature;

uniform mat4 uProjection;
uniform mat4 uView;

out vec3 vWorldPosition;
out vec3 vNormal;
out vec3 vColor;
out float vConfidence;
out float vDeviation;
out float vFeature;

void main() {
  vWorldPosition = aPosition;
  vNormal = aNormal;
  vColor = aColor;
  vConfidence = aConfidence;
  vDeviation = aDeviation;
  vFeature = aFeature;
  gl_Position = uProjection * uView * vec4(aPosition, 1.0);
}
`;

export const meshFragment = `#version 300 es
precision highp float;

in vec3 vWorldPosition;
in vec3 vNormal;
in vec3 vColor;
in float vConfidence;
in float vDeviation;
in float vFeature;

uniform vec3 uCameraPosition;
uniform vec2 uResolution;
uniform float uMode;
uniform float uOverlay;
uniform float uErrorScale;
uniform float uContours;

out vec4 outColor;

void main() {
  vec3 N = normalize(vNormal);
  vec3 V = normalize(uCameraPosition - vWorldPosition);
  vec3 L1 = normalize(vec3(-0.46, 0.92, 0.50));
  vec3 L2 = normalize(vec3(0.72, 0.32, -0.58));
  float key = max(dot(N, L1), 0.0);
  float fill = max(dot(N, L2), 0.0);
  float facing = max(dot(N, V), 0.0);
  float rim = pow(1.0 - facing, 2.4);
  float spec = pow(max(dot(normalize(L1 + V), N), 0.0), 54.0);
  float shade = 0.145 + key * 0.62 + fill * 0.19 + rim * 0.105 + spec * 0.45;
  float ao = 0.80 + 0.20 * clamp(N.y * 0.5 + 0.5, 0.0, 1.0);
  vec3 neutral = vec3(0.78, 0.85, 0.89) * shade * ao;
  neutral += vec3(0.16, 0.19, 0.21) * pow(1.0 - facing, 3.0);

  float lum = dot(neutral, vec3(0.2126, 0.7152, 0.0722));
  vec3 tint = vColor * (0.34 + lum * 0.94);
  float cavity = (vFeature > 2.5 && vFeature < 3.5) ? 0.58 : 1.0;
  tint *= cavity;
  vec3 classColor = mix(neutral, tint, uOverlay);
  float softConfidence = mix(0.28, 1.0, clamp(vConfidence, 0.0, 1.0));
  classColor *= mix(0.84, 1.0, softConfidence);

  float signedError = clamp(vDeviation / max(uErrorScale, 0.0001), -1.0, 1.0);
  vec3 cool = vec3(0.23, 0.50, 0.94);
  vec3 warm = vec3(1.00, 0.43, 0.20);
  vec3 nominal = vec3(0.54, 0.59, 0.59);
  vec3 errorHue = signedError < 0.0
    ? mix(nominal, cool, -signedError)
    : mix(nominal, warm, signedError);
  vec3 auditColor = errorHue * (0.40 + lum * 0.88);
  float scaled = signedError * 5.0;
  float iso = abs(fract(scaled + 0.5) - 0.5);
  float width = max(fwidth(iso), 0.0025);
  float line = (1.0 - smoothstep(0.0, width * 1.35, iso)) * uContours;
  line *= smoothstep(0.01, 0.1, abs(signedError));
  auditColor = mix(auditColor, vec3(0.88, 0.96, 1.0), line * 0.42);

  vec3 finalColor;
  if (uMode < 0.5) {
    finalColor = classColor;
  } else if (uMode < 1.5) {
    finalColor = auditColor;
  } else {
    float x = gl_FragCoord.x / uResolution.x;
    float y = gl_FragCoord.y / uResolution.y;
    float side = x + (1.0 - y) * 0.30;
    float seam = smoothstep(0.003, 0.0, abs(side - 0.61));
    finalColor = side < 0.61 ? classColor : auditColor;
    finalColor = mix(finalColor, vec3(0.70, 0.93, 1.0), seam * 0.75);
  }

  outColor = vec4(finalColor, 1.0);
}
`;

export const lineVertex = `#version 300 es
precision highp float;
in vec3 aPosition;
in vec3 aOther;
in vec3 aNormal;
in float aSide;
in float aConfidence;
in float aStyle;

uniform mat4 uProjection;
uniform mat4 uView;
uniform vec2 uResolution;

out float vConfidence;
out float vStyle;

void main() {
  vec4 selfClip = uProjection * uView * vec4(aPosition + aNormal * 0.17, 1.0);
  vec4 otherClip = uProjection * uView * vec4(aOther + aNormal * 0.17, 1.0);
  vec2 selfNdc = selfClip.xy / max(selfClip.w, 0.0001);
  vec2 otherNdc = otherClip.xy / max(otherClip.w, 0.0001);
  vec2 dir = otherNdc - selfNdc;
  if (length(dir) < 0.00001) {
    gl_Position = selfClip;
    vConfidence = aConfidence;
    vStyle = aStyle;
    return;
  }
  vec2 perp = vec2(-dir.y, dir.x);
  float widthPx = 0.82 + 1.1 * clamp(aConfidence, 0.0, 1.0);
  selfClip.xy += normalize(perp) * aSide * vec2(
    2.0 * widthPx / uResolution.x,
    2.0 * widthPx / uResolution.y
  ) * selfClip.w;
  gl_Position = selfClip;
  vConfidence = aConfidence;
  vStyle = aStyle;
}
`;

export const lineFragment = `#version 300 es
precision highp float;
in float vConfidence;
in float vStyle;
out vec4 outColor;

void main() {
  vec3 low = vec3(1.0, 0.42, 0.16);
  vec3 high = vec3(0.73, 0.95, 1.0);
  vec3 color = mix(low, high, clamp(vConfidence, 0.0, 1.0));
  outColor = vec4(color, (0.50 + vConfidence * 0.40) * vStyle);
}
`;
