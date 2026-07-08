import WorkflowTable from '@/components/workflow/WorkflowTable';

export default function WorkflowTodoPage() {
  return (
    <WorkflowTable
      scope="todo"
      title="待审批工单"
      assetBasePath="/admin"
      emptyDescription="暂无待审批工单"
    />
  );
}
