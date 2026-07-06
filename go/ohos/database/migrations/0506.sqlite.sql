-- 对齐 48 RSS 的 tb_sys_configs 补丁链:
--   tb_sys_config_20250218        FVisible 分组(系统批=2 → 用户配置页;用户批=1 → 配置设置页)
--   tb_sys_configspath_plc        MaxSpeed=660 / MaxRealWeightCount=66 种子值
--   tb_sys_config_setfvaluetype_20250219  两项 FValueType=3(数字)
--   tb_sys_configs_fzhtype20250326        FZhType 中文显示名

-- 1) 系统批 → FVisible=2(48 UserConfigButton / ConfigSetForm(2))
UPDATE "tb_sys_configs" SET "FVisible" = 2
WHERE "FModuleName" = 'RSS' AND "FType" IN (
  'BaseUrl', '绿萌大屏', 'NetworkUrl', 'AppKey', 'AppSecret',
  'PLC服务端IP', 'PLC服务端口', '故障上传', '清空出口', 'IsSystemErr',
  'IsOfflineParam', 'IsExitScreenDisplay', 'isExitScreenDisplay', 'IsUserManagement'
);

-- 2) 用户批 → FVisible=1(48 配置设置按钮 / ConfigSetForm(1);含 48 补丁中的拼写变体)
UPDATE "tb_sys_configs" SET "FVisible" = 1
WHERE "FModuleName" = 'RSS' AND "FType" IN (
  '拆分器距离', '拆分器距离2', '出口垂直滚动条', '大屏开关', '大屏标题', '大屏端口',
  'MaxRealWeightCount', 'MaxSpeed', 'ACS-SIM', 'SIM-PLCIP', 'SIM', '自定义单价',
  'SumWeightUnit', 'SumWeightAccuracy', 'WeightAccuracy', 'SizeAccuracy', 'DensityAccuracy',
  'CheckExport', 'BatchWeightUnit', 'FruitSizeUnit', 'FruitWeightUnit', 'GradeSizeUnit',
  'GradeWeightUnit', 'ProcessInfoWeightUnit',
  'GradeAccuracy', 'GradeAccracy', 'BatchWeightAccuracy', 'BatchWeightAccracy',
  'ProcessInfoWeightAccuracy', 'ProcessInfoWeightAccracy'
);

-- 3) 缺行则按 48 path_plc 种子补齐
INSERT INTO "tb_sys_configs" ("FType", "FValue", "FModuleName", "FVisible", "FEnType", "FValueType", "FValueTypeDetail", "FSubSystem", "FCreateDate")
SELECT 'MaxSpeed', '660', 'RSS', 1, 'MaxSpeed', 3, NULL, 0, '2026-07-05 00:00:00.000000'
WHERE NOT EXISTS (SELECT 1 FROM "tb_sys_configs" WHERE "FModuleName" = 'RSS' AND "FType" = 'MaxSpeed');

INSERT INTO "tb_sys_configs" ("FType", "FValue", "FModuleName", "FVisible", "FEnType", "FValueType", "FValueTypeDetail", "FSubSystem", "FCreateDate")
SELECT 'MaxRealWeightCount', '66', 'RSS', 1, 'MaxRealWeightCount', 3, NULL, 0, '2026-07-05 00:00:00.000000'
WHERE NOT EXISTS (SELECT 1 FROM "tb_sys_configs" WHERE "FModuleName" = 'RSS' AND "FType" = 'MaxRealWeightCount');

-- 4) 空值补 48 种子值(不覆盖现场已调整的值)
UPDATE "tb_sys_configs" SET "FValue" = '660'
WHERE "FModuleName" = 'RSS' AND "FType" = 'MaxSpeed' AND ("FValue" IS NULL OR "FValue" = '');

UPDATE "tb_sys_configs" SET "FValue" = '66'
WHERE "FModuleName" = 'RSS' AND "FType" = 'MaxRealWeightCount' AND ("FValue" IS NULL OR "FValue" = '');

-- 5) FValueType=3(数字输入)
UPDATE "tb_sys_configs" SET "FValueType" = 3
WHERE "FModuleName" = 'RSS' AND "FType" IN ('MaxSpeed', 'MaxRealWeightCount');

-- 6) FZhType 中文显示名(仅补空,清单来自 48 fzhtype20250326 补丁)
UPDATE "tb_sys_configs" SET "FZhType" = '最大速度' WHERE "FModuleName" = 'RSS' AND "FType" = 'MaxSpeed' AND ("FZhType" IS NULL OR "FZhType" = '');
UPDATE "tb_sys_configs" SET "FZhType" = '最大产量' WHERE "FModuleName" = 'RSS' AND "FType" = 'MaxRealWeightCount' AND ("FZhType" IS NULL OR "FZhType" = '');
UPDATE "tb_sys_configs" SET "FZhType" = '报表统计数据总重量单位' WHERE "FModuleName" = 'RSS' AND "FType" = 'SumWeightUnit' AND ("FZhType" IS NULL OR "FZhType" = '');
UPDATE "tb_sys_configs" SET "FZhType" = '报表统计数据总重量精度' WHERE "FModuleName" = 'RSS' AND "FType" = 'SumWeightAccuracy' AND ("FZhType" IS NULL OR "FZhType" = '');
UPDATE "tb_sys_configs" SET "FZhType" = '出口屏开关' WHERE "FModuleName" = 'RSS' AND "FType" = 'IsExitScreenDisplay' AND ("FZhType" IS NULL OR "FZhType" = '');
UPDATE "tb_sys_configs" SET "FZhType" = '等级重量精度' WHERE "FModuleName" = 'RSS' AND "FType" = 'GradeWeightAccuracy' AND ("FZhType" IS NULL OR "FZhType" = '');
UPDATE "tb_sys_configs" SET "FZhType" = '等级尺寸精度' WHERE "FModuleName" = 'RSS' AND "FType" = 'GradeSizeAccuracy' AND ("FZhType" IS NULL OR "FZhType" = '');
UPDATE "tb_sys_configs" SET "FZhType" = '等级密度精度' WHERE "FModuleName" = 'RSS' AND "FType" = 'GradeDensityAccuracy' AND ("FZhType" IS NULL OR "FZhType" = '');
UPDATE "tb_sys_configs" SET "FZhType" = '等级重量的单位' WHERE "FModuleName" = 'RSS' AND "FType" = 'GradeWeightUnit' AND ("FZhType" IS NULL OR "FZhType" = '');
UPDATE "tb_sys_configs" SET "FZhType" = '等级尺寸单位' WHERE "FModuleName" = 'RSS' AND "FType" = 'GradeSizeUnit' AND ("FZhType" IS NULL OR "FZhType" = '');
UPDATE "tb_sys_configs" SET "FZhType" = '报表等级尺寸精度' WHERE "FModuleName" = 'RSS' AND "FType" = 'ReportSizeAccuracy' AND ("FZhType" IS NULL OR "FZhType" = '');
UPDATE "tb_sys_configs" SET "FZhType" = '报表等级重量精度' WHERE "FModuleName" = 'RSS' AND "FType" = 'ReportWeightAccuracy' AND ("FZhType" IS NULL OR "FZhType" = '');
UPDATE "tb_sys_configs" SET "FZhType" = '报表等级重量单位' WHERE "FModuleName" = 'RSS' AND "FType" = 'ReportWeightUnit' AND ("FZhType" IS NULL OR "FZhType" = '');
UPDATE "tb_sys_configs" SET "FZhType" = '报表等级尺寸单位' WHERE "FModuleName" = 'RSS' AND "FType" = 'ReportSizeUint' AND ("FZhType" IS NULL OR "FZhType" = '');
UPDATE "tb_sys_configs" SET "FZhType" = '报表总重量单位' WHERE "FModuleName" = 'RSS' AND "FType" = 'ReportSumWeightUnit' AND ("FZhType" IS NULL OR "FZhType" = '');
UPDATE "tb_sys_configs" SET "FZhType" = '报表总重量精度' WHERE "FModuleName" = 'RSS' AND "FType" = 'ReportSumWeightAccuracy' AND ("FZhType" IS NULL OR "FZhType" = '');
UPDATE "tb_sys_configs" SET "FZhType" = '主页侧边栏是否只显示图标' WHERE "FModuleName" = 'RSS' AND "FType" = 'MainFormIconOnly' AND ("FZhType" IS NULL OR "FZhType" = '');
UPDATE "tb_sys_configs" SET "FZhType" = '统计数据箱重单位' WHERE "FModuleName" = 'RSS' AND "FType" = 'ReportBoxWeightUint' AND ("FZhType" IS NULL OR "FZhType" = '');
