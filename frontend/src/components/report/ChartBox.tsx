import { Spin } from 'antd';
import { useEffect, useRef, useState, type ReactElement } from 'react';

interface ChartBoxProps {
  height?: number;
  children: (width: number) => ReactElement | null;
}

/**
 * 在可见容器内测量宽度后再渲染 G2 图表，避免 Tabs 切换后 autoFit 偏移。
 */
export default function ChartBox({ height = 300, children }: ChartBoxProps) {
  const ref = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState(0);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;

    const sync = () => {
      const next = Math.floor(el.getBoundingClientRect().width);
      if (next > 0) setWidth(next);
    };

    sync();
    const ro = new ResizeObserver(sync);
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  return (
    <div ref={ref} style={{ width: '100%', height, minHeight: height }}>
      {width > 0 ? (
        children(width)
      ) : (
        <div
          style={{
            height,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          <Spin />
        </div>
      )}
    </div>
  );
}
