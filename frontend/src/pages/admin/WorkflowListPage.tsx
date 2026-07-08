import WorkflowTable from '@/components/workflow/WorkflowTable';

export default function WorkflowListPage() {
  return (
    <WorkflowTable
      scope="all"
      title="全部工单"
      assetBasePath="/admin"
      emptyDescription="暂无工单记录"
    />
  );
}
