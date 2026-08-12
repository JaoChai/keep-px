-- ทิ้งระบบ auth เดิมของเรา (JWT + refresh token + Google OAuth ตรง ๆ)
-- ตั้งแต่ย้ายไปใช้ Neon Auth ใน Task 3-5
-- การ DROP COLUMN google_id จะลบ partial unique index idx_customers_google_id
-- ที่อ้างถึง column นั้นออกให้อัตโนมัติใน PostgreSQL
DROP TABLE IF EXISTS refresh_tokens;
ALTER TABLE customers DROP COLUMN IF EXISTS google_id;
