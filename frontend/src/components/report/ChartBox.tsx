import type { ReactNode } from 'react';

interface ChartBoxProps {
  height?: number;
  children: ReactNode;
}

/** 约束图表容器宽度，避免 G2 autoFit 在 flex 布局下偏移 */
export default function ChartBox({ height = 300, children }: ChartBoxProps) {
  return (
    <div
      style={{
        width: '100%',
        height,
        position: 'relative',
        overflow: 'hidden',
      }}
    >
      {children}
    </div>
  );
}
