import WorkflowTable from '@/components/workflow/WorkflowTable';

export default function WorkflowListPage() {
  return (
    <WorkflowTable
      scope="all"
      title="本院全部工单"
      assetBasePath="/college"
      emptyDescription="暂无本院工单记录"
    />
  );
}
