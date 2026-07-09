import { Result, Skeleton } from 'antd';
import { useEffect, useMemo, useRef, useState } from 'react';
import {
  FUniver,
  LocaleType,
  Univer,
} from '@univerjs/core';
import { defaultTheme } from '@univerjs/design';
import DesignZhCN from '@univerjs/design/locale/zh-CN';
import { UniverDocsPlugin } from '@univerjs/docs';
import { UniverDocsUIPlugin } from '@univerjs/docs-ui';
import DocsUIZhCN from '@univerjs/docs-ui/locale/zh-CN';
import { UniverRenderEnginePlugin } from '@univerjs/engine-render';
import { UniverSheetsPlugin } from '@univerjs/sheets';
import { UniverSheetsUIPlugin } from '@univerjs/sheets-ui';
import SheetsZhCN from '@univerjs/sheets/locale/zh-CN';
import SheetsUIZhCN from '@univerjs/sheets-ui/locale/zh-CN';
import { UniverUIPlugin } from '@univerjs/ui';
import UIZhCN from '@univerjs/ui/locale/zh-CN';
import '@univerjs/design/lib/index.css';
import '@univerjs/ui/lib/index.css';
import '@univerjs/sheets-ui/lib/index.css';
import '@univerjs/sheets/facade';
import type { SpreadsheetRow } from '@/utils/inventory';
import { buildInventoryWorkbook, readRowsFromWorkbook } from '@/components/inventory/univerWorkbook';
import InventorySpreadsheet from '@/components/inventory/InventorySpreadsheet';

interface UniverSpreadsheetProps {
  rows: SpreadsheetRow[];
  readOnly: boolean;
  onChange: (rows: SpreadsheetRow[]) => void;
}

export default function UniverSpreadsheet({ rows, readOnly, onChange }: UniverSpreadsheetProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const univerRef = useRef<Univer | null>(null);
  const apiRef = useRef<FUniver | null>(null);
  const rowsRef = useRef(rows);
  const syncingRef = useRef(false);
  const [initError, setInitError] = useState<string | null>(null);
  const [ready, setReady] = useState(false);

  rowsRef.current = rows;
  const structureKey = useMemo(() => rows.map((row) => row.key).join('|'), [rows]);

  useEffect(() => {
    if (!rows.length) return;

    const container = containerRef.current;
    if (!container) return;

    let disposed = false;
    setInitError(null);
    setReady(false);

    try {
      const univer = new Univer({
        theme: defaultTheme,
        locale: LocaleType.ZH_CN,
        locales: {
          [LocaleType.ZH_CN]: {
            ...DesignZhCN,
            ...UIZhCN,
            ...SheetsZhCN,
            ...SheetsUIZhCN,
            ...DocsUIZhCN,
          },
        },
      });

      univer.registerPlugin(UniverRenderEnginePlugin);
      univer.registerPlugin(UniverUIPlugin, {
        container,
        header: false,
        toolbar: false,
        footer: false,
        contextMenu: !readOnly,
      });
      univer.registerPlugin(UniverDocsPlugin);
      univer.registerPlugin(UniverDocsUIPlugin);
      univer.registerPlugin(UniverSheetsPlugin);
      univer.registerPlugin(UniverSheetsUIPlugin, { formulaBar: false });

      const api = FUniver.newAPI(univer);
      api.createWorkbook(buildInventoryWorkbook(rowsRef.current));

      const syncFromSheet = () => {
        if (readOnly || syncingRef.current) return;
        const workbook = api.getActiveWorkbook();
        const sheet = workbook?.getActiveSheet();
        if (!sheet) return;
        const next = readRowsFromWorkbook(rowsRef.current, (row, col) =>
          sheet.getRange(row, col).getValue(),
        );
        onChange(next);
      };

      const commandDisposable = api.addEvent(api.Event.CommandExecuted, () => {
        syncFromSheet();
      });

      univerRef.current = univer;
      apiRef.current = api;
      if (!disposed) setReady(true);

      return () => {
        disposed = true;
        commandDisposable.dispose();
        univer.dispose();
        univerRef.current = null;
        apiRef.current = null;
        container.innerHTML = '';
      };
    } catch (err) {
      setInitError(err instanceof Error ? err.message : 'Univer 初始化失败');
      return undefined;
    }
  }, [readOnly, onChange, rows.length]);

  useEffect(() => {
    const api = apiRef.current;
    if (!api || !ready || !rows.length) return;

    syncingRef.current = true;
    api.disposeUnit('inventory-workbook');
    api.createWorkbook(buildInventoryWorkbook(rowsRef.current));
    syncingRef.current = false;
  }, [structureKey, ready, rows.length]);

  if (!rows.length) {
    return <Skeleton active paragraph={{ rows: 10 }} />;
  }

  if (initError) {
    return (
      <>
        <Result
          status="warning"
          title="Univer 表格加载失败"
          subTitle={initError}
          extra={<span>已回退到基础表格模式</span>}
        />
        <InventorySpreadsheet rows={rows} readOnly={readOnly} onChange={onChange} />
      </>
    );
  }

  return (
    <div style={{ minHeight: 520 }}>
      {!ready ? <Skeleton active paragraph={{ rows: 10 }} /> : null}
      <div
        ref={containerRef}
        style={{
          width: '100%',
          height: 'calc(100vh - 280px)',
          minHeight: 480,
          display: ready ? 'block' : 'none',
        }}
      />
    </div>
  );
}
