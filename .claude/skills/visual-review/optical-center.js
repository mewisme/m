#!/usr/bin/env node

// Compute a simple alpha-weighted visual centroid for RGBA pixel data supplied as JSON.
// Input: {"width":N,"height":N,"rgba":[r,g,b,a,...]}
// Output: {"x":...,"y":...,"offsetX":...,"offsetY":...}

let input = "";
process.stdin.setEncoding("utf8");
process.stdin.on("data", (chunk) => (input += chunk));
process.stdin.on("end", () => {
  const { width, height, rgba } = JSON.parse(input);
  let total = 0;
  let sx = 0;
  let sy = 0;
  for (let y = 0; y < height; y += 1) {
    for (let x = 0; x < width; x += 1) {
      const alpha = rgba[(y * width + x) * 4 + 3] / 255;
      total += alpha;
      sx += x * alpha;
      sy += y * alpha;
    }
  }
  const cx = total === 0 ? (width - 1) / 2 : sx / total;
  const cy = total === 0 ? (height - 1) / 2 : sy / total;
  process.stdout.write(`${JSON.stringify({
    x: cx,
    y: cy,
    offsetX: cx - (width - 1) / 2,
    offsetY: cy - (height - 1) / 2,
  })}\n`);
});
