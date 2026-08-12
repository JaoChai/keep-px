import { describe, it, expect } from "vitest";
import { normalizeClientIP } from "./client-ip";

function req(headers: Record<string, string>): Request {
  return new Request("https://example.com/api/v1/events/ingest", { headers });
}

describe("normalizeClientIP", () => {
  it("เก็บ CF-Connecting-IP ที่ Cloudflare เซ็ตไว้", () => {
    const out = normalizeClientIP(req({ "CF-Connecting-IP": "203.0.113.7" }));
    expect(out.headers.get("CF-Connecting-IP")).toBe("203.0.113.7");
  });

  it("ลบ header ที่ client ปลอมได้ทุกตัวทิ้ง", () => {
    const out = normalizeClientIP(
      req({
        "CF-Connecting-IP": "203.0.113.7",
        "X-Real-IP": "198.51.100.1",
        "X-Forwarded-For": "192.0.2.1",
        "True-Client-IP": "192.0.2.2",
        "X-Forwarded-Proto": "http",
      }),
    );
    expect(out.headers.get("X-Real-IP")).toBeNull();
    expect(out.headers.get("X-Forwarded-For")).toBeNull();
    expect(out.headers.get("True-Client-IP")).toBeNull();
    expect(out.headers.get("X-Forwarded-Proto")).toBeNull();
  });

  it("ถ้าไม่มี CF-Connecting-IP ต้องไม่หลง trust ค่าปลอม", () => {
    const out = normalizeClientIP(req({ "X-Real-IP": "198.51.100.1" }));
    expect(out.headers.get("CF-Connecting-IP")).toBeNull();
    expect(out.headers.get("X-Real-IP")).toBeNull();
  });

  it("ไม่แตะ header อื่น", () => {
    const out = normalizeClientIP(
      req({ "CF-Connecting-IP": "203.0.113.7", Authorization: "Bearer abc" }),
    );
    expect(out.headers.get("Authorization")).toBe("Bearer abc");
  });
});
