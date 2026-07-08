import { useEffect, useMemo, useState } from 'react';
import { Button, Card, DatePicker, Form, Input, Select, Space, message } from 'antd';
import dayjs from 'dayjs';
import { useNavigate } from 'react-router-dom';
import { useCreateTaskMutation } from '@/store/api/inventoryApi';
import { useGetDeptTreeQuery, useListUsersQuery } from '@/store/api/userApi';
import { useAppSelector } from '@/store/hooks';
import { selectCurrentUser } from '@/store/slices/authSlice';

interface InventoryTaskFormProps {
  basePath: '/admin' | '/college';
}

function flattenDeptTree(
  nodes: { id: number; deptName: string; children: typeof nodes | null }[],
): { label: string; value: number }[] {
  const result: { label: string; value: number }[] = [];
  const walk = (list: typeof nodes, prefix = '') => {
    for (const node of list) {
      const label = prefix ? `${prefix} / ${node.deptName}` : node.deptName;
      result.push({ label, value: node.id });
      if (node.children?.length) walk(node.children, label);
    }
  };
  walk(nodes);
  return result;
}

export default function InventoryTaskForm({ basePath }: InventoryTaskFormProps) {
  const navigate = useNavigate();
  const user = useAppSelector(selectCurrentUser);
  const [form] = Form.useForm();
  const [scopeDeptId, setScopeDeptId] = useState<number>();

  const { data: deptTree } = useGetDeptTreeQuery();
  const { data: usersData } = useListUsersQuery(
    { page: 1, pageSize: 100, departmentId: scopeDeptId, roleLevel: 3 },
    { skip: !scopeDeptId },
  );
  const [createTask, { isLoading }] = useCreateTaskMutation();

  const deptOptions = useMemo(
    () => (deptTree ? flattenDeptTree(deptTree) : []),
    [deptTree],
  );

  const userOptions = useMemo(
    () =>
      (usersData?.list ?? []).map((u) => ({
        label: `${u.realName} (${u.username})`,
        value: u.id,
      })),
    [usersData?.list],
  );

  useEffect(() => {
    if (user?.roleLevel === 2 && user.departmentId) {
      form.setFieldValue('scopeDeptId', user.departmentId);
      setScopeDeptId(user.departmentId);
    }
  }, [user, form]);

  const onFinish = async (values: {
    taskName: string;
    scopeDeptId: number;
    timeRange: [dayjs.Dayjs, dayjs.Dayjs];
    assigneeIds: number[];
  }) => {
    try {
      const result = await createTask({
        taskName: values.taskName,
        scopeDeptId: values.scopeDeptId,
        startTime: values.timeRange[0].toISOString(),
        endTime: values.timeRange[1].toISOString(),
        assigneeIds: values.assigneeIds,
      }).unwrap();
      message.success('盘点任务已创建');
      navigate(`${basePath}/inventory/tasks/${result.id}`);
    } catch (err: unknown) {
      const e = err as { message?: string };
      message.error(e.message ?? '创建失败');
    }
  };

  return (
    <Card title="创建盘点任务">
      <Form
        form={form}
        layout="vertical"
        onFinish={onFinish}
        style={{ maxWidth: 560 }}
        initialValues={{
          timeRange: [dayjs().startOf('day'), dayjs().add(7, 'day').endOf('day')],
        }}
      >
        <Form.Item
          name="taskName"
          label="任务名称"
          rules={[{ required: true, message: '请输入任务名称' }, { max: 50 }]}
        >
          <Input placeholder="2026 信息学院实验室盘点" />
        </Form.Item>
        <Form.Item
          name="scopeDeptId"
          label="盘点范围"
          rules={[{ required: true, message: '请选择盘点范围' }]}
        >
          <Select
            options={deptOptions}
            showSearch
            optionFilterProp="label"
            disabled={user?.roleLevel === 2}
            onChange={(v) => {
              setScopeDeptId(v);
              form.setFieldValue('assigneeIds', []);
            }}
          />
        </Form.Item>
        <Form.Item
          name="timeRange"
          label="时间窗"
          rules={[{ required: true, message: '请选择时间范围' }]}
        >
          <DatePicker.RangePicker showTime style={{ width: '100%' }} />
        </Form.Item>
        <Form.Item
          name="assigneeIds"
          label="指派盘点员"
          rules={[{ required: true, message: '请至少选择一名盘点员' }]}
        >
          <Select mode="multiple" options={userOptions} placeholder="先选择盘点范围" />
        </Form.Item>
        <Form.Item>
          <Space>
            <Button type="primary" htmlType="submit" loading={isLoading}>
              创建
            </Button>
            <Button onClick={() => navigate(`${basePath}/inventory/tasks`)}>取消</Button>
          </Space>
        </Form.Item>
      </Form>
    </Card>
  );
}
