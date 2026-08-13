import { describe, it, expect } from "vitest";
import { applySecurityHeaders } from "./headers";

function headersFor(pathname: string): Headers {
  return applySecurityHeaders(new Response("x"), pathname).headers;
}

describe("applySecurityHeaders", () => {
  it("ใส่ครบทั้ง 7 ตัวสำหรับ dashboard", () => {
    const h = headersFor("/dashboard");
    expect(h.get("X-Content-Type-Options")).toBe("nosniff");
    expect(h.get("X-Frame-Options")).toBe("DENY");
    expect(h.get("Referrer-Policy")).toBe("strict-origin-when-cross-origin");
    expect(h.get("Permissions-Policy")).toBe("camera=(), microphone=(), geolocation=()");
    expect(h.get("Strict-Transport-Security")).toBe("max-age=31536000; includeSubDomains");
    expect(h.get("Cross-Origin-Opener-Policy")).toBe("same-origin");
    expect(h.get("Content-Security-Policy")).toContain("default-src 'self'");
  });

  it("sale page ต้องไม่มี X-Frame-Options เพราะต้องฝังในหน้าอื่นได้", () => {
    const h = headersFor("/p/my-slug");
    expect(h.get("X-Frame-Options")).toBeNull();
    expect(h.get("Permissions-Policy")).toBeNull();
    expect(h.get("Cross-Origin-Opener-Policy")).toBeNull();
    expect(h.get("X-Content-Type-Options")).toBe("nosniff");
    expect(h.get("Referrer-Policy")).toBe("strict-origin-when-cross-origin");
    expect(h.get("Strict-Transport-Security")).toBe("max-age=31536000; includeSubDomains");
  });

  it("sale page CSP อนุญาต img จากทุกที่ แต่ dashboard ไม่", () => {
    expect(headersFor("/p/x").get("Content-Security-Policy")).toContain("img-src * data:");
    expect(headersFor("/dashboard").get("Content-Security-Policy")).toContain(
      "img-src 'self' data: *.googleusercontent.com *.r2.dev",
    );
  });

  it("CSP ต้องไม่มี BACKEND_URL placeholder หลงเหลือ", () => {
    for (const p of ["/dashboard", "/p/x"]) {
      expect(headersFor(p).get("Content-Security-Policy")).not.toContain("${");
    }
  });

  it("ไม่ทำลาย header เดิมของ response", () => {
    const res = new Response("x", { headers: { "Content-Type": "text/html" } });
    expect(applySecurityHeaders(res, "/").headers.get("Content-Type")).toBe("text/html");
  });

  it("CSP ยอมให้ต่อ Neon Auth และไม่เหลือ Google Sign-In", () => {
    const csp = headersFor("/dashboard").get("Content-Security-Policy")!;
    expect(csp).toContain("*.neon.tech");
    expect(csp).not.toContain("accounts.google.com");
  });

  it("CSP ยอมให้โหลด Cloudflare Web Analytics beacon ทั้ง dashboard และ sale page", () => {
    for (const p of ["/dashboard", "/p/x"]) {
      expect(headersFor(p).get("Content-Security-Policy")).toContain(
        "static.cloudflareinsights.com",
      );
    }
  });
});
