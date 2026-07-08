import WorkflowTable from '@/components/workflow/WorkflowTable';

export default function WorkflowMyPage() {
  return (
    <WorkflowTable
      scope="my"
      title="我的工单"
      assetBasePath="/user"
      emptyDescription="暂无申请记录"
      showCreateLink
    />
  );
}
