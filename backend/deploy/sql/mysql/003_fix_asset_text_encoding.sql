-- 修复 asset_ledger 中文乱码（使用 UNHEX 写入正确 UTF-8 字节，避免客户端编码干扰）
-- 执行：docker exec -i fams-mysql /usr/bin/mysql -ufams -pfams_dev_pass fams_asset < deploy/sql/mysql/003_fix_asset_text_encoding.sql

UPDATE asset_ledger SET name=UNHEX('e6bf80e58589e58887e589b2e69cba'), category=UNHEX('e8aebee5a487'), location=UNHEX('e4b880e58fb7e5ae9ee9aa8ce6a5bc313031') WHERE id=501;
UPDATE asset_ledger SET name=UNHEX('3344e68993e58db0e69cba'), category=UNHEX('e8aebee5a487'), location=UNHEX('e4b880e58fb7e5ae9ee9aa8ce6a5bc313032') WHERE id=502;
UPDATE asset_ledger SET name=UNHEX('e5ae9ee9aa8ce58fb0'), category=UNHEX('e5aeb6e585b7'), location=UNHEX('e8bdafe4bbb6e5b7a5e7a88be5ae9ee9aa8ce5aea4') WHERE id=503;
UPDATE asset_ledger SET name=UNHEX('e7a4bae6b3a2e599a8'), category=UNHEX('e8aebee5a487'), location=UNHEX('e7bd91e7bb9ce5b7a5e7a88be5ae9ee9aa8ce5aea4') WHERE id=504;
UPDATE asset_ledger SET name=UNHEX('e8bda6e5ba8a'), category=UNHEX('e8aebee5a487'), location=UNHEX('e69cbae6a2b0e5b7a5e7a88be5ada6e999a2') WHERE id=505;
