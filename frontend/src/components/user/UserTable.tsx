import { useCallback, useMemo, useState } from 'react';
import { Button, Card, Input, Modal, Result, Space, Switch, Tag, message } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { ProTable, type ProColumns } from '@ant-design/pro-components';
import type { UserListItem } from '@/types/api';
import {
  useForceLogoutMutation,
  useGetCollegeSubtreeQuery,
  useListUsersQuery,
  useUpdateUserStatusMutation,
} from '@/store/api/userApi';
import CreateUserModal from '@/components/user/CreateUserModal';
import { ROLE_MAP } from '@/utils/constants';

interface UserTableProps {
  roleLevel?: 1 | 2;
  restrictDeptId?: number;
}

export default function UserTable({ roleLevel = 1, restrictDeptId }: UserTableProps) {
  const [keyword, setKeyword] = useState('');
  const [searchKeyword, setSearchKeyword] = useState<string | undefined>();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [createOpen, setCreateOpen] = useState(false);
  const [updatingId, setUpdatingId] = useState<number | null>(null);

  const { data: subtree } = useGetCollegeSubtreeQuery(undefined, {
    skip: roleLevel !== 2 || !restrictDeptId,
  });

  const { data, isLoading, isError, refetch } = useListUsersQuery({
    page,
    pageSize,
    keyword: searchKeyword,
  });
  const [updateStatus] = useUpdateUserStatusMutation();
  const [forceLogout, { isLoading: forcing }] = useForceLogoutMutation();

  const allowedDeptIds = useMemo(() => {
    if (roleLevel !== 2 || !subtree?.deptIds?.length) return null;
    return new Set(subtree.deptIds);
  }, [roleLevel, subtree?.deptIds]);

  const tableData = useMemo(() => {
    const list = data?.list ?? [];
    if (!allowedDeptIds) return list;
    return list.filter((u) => allowedDeptIds.has(u.departmentId));
  }, [data?.list, allowedDeptIds]);

  const isUserInScope = useCallback(
    (row: UserListItem) => !allowedDeptIds || allowedDeptIds.has(row.departmentId),
    [allowedDeptIds],
  );

  const columns: ProColumns<UserListItem>[] = useMemo(
    () => [
      { title: '用户名', dataIndex: 'username', width: 140 },
      { title: '姓名', dataIndex: 'realName', width: 120 },
      {
        title: '角色',
        dataIndex: 'roleLevel',
        width: 120,
        render: (_, r) => <Tag>{ROLE_MAP[r.roleLevel] ?? `角色${r.roleLevel}`}</Tag>,
      },
      {
        title: '部门',
        dataIndex: 'departmentName',
        width: 160,
        ellipsis: true,
        render: (_, r) => r.departmentName ?? `#${r.departmentId}`,
      },
      {
        title: '状态',
        dataIndex: 'status',
        width: 100,
        render: (_, r) => (
          <Switch
            checked={r.status === 1}
            checkedChildren="启用"
            unCheckedChildren="禁用"
            disabled={!isUserInScope(r)}
            loading={updatingId === r.id}
            onChange={async (checked) => {
              setUpdatingId(r.id);
              try {
                await updateStatus({
                  id: r.id,
                  body: { status: checked ? 1 : 0 },
                }).unwrap();
                message.success(checked ? '已启用' : '已禁用');
              } catch (err: unknown) {
                const e = err as { message?: string };
                message.error(e.message ?? '状态更新失败');
              } finally {
                setUpdatingId(null);
              }
            }}
          />
        ),
      },
      {
        title: '操作',
        width: 120,
        render: (_, r) => (
          <Button
            type="link"
            danger
            disabled={forcing || !isUserInScope(r)}
            onClick={() => {
              Modal.confirm({
                title: '强制下线',
                content: `确定强制 ${r.realName}（${r.username}）下线？`,
                okText: '确认',
                cancelText: '取消',
                onOk: async () => {
                  try {
                    await forceLogout(r.id).unwrap();
                    message.success('已强制下线');
                  } catch (err: unknown) {
                    const e = err as { message?: string };
                    message.error(e.message ?? '操作失败');
                  }
                },
              });
            }}
          >
            强制下线
          </Button>
        ),
      },
    ],
    [forceLogout, forcing, updateStatus, updatingId, isUserInScope],
  );

  const onSearch = () => {
    setPage(1);
    setSearchKeyword(keyword || undefined);
  };

  if (isError) {
    return (
      <Result
        status="error"
        title="用户列表加载失败"
        extra={
          <Button type="primary" onClick={() => refetch()}>
            重试
          </Button>
        }
      />
    );
  }

  return (
    <>
      <Card
        title="用户管理"
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
            创建用户
          </Button>
        }
      >
        <Space style={{ marginBottom: 16 }}>
          <Input
            placeholder="搜索用户名或姓名"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            onPressEnter={onSearch}
            style={{ width: 240 }}
            allowClear
          />
          <Button type="primary" onClick={onSearch}>
            搜索
          </Button>
        </Space>

        <ProTable<UserListItem>
          rowKey="id"
          search={false}
          options={false}
          loading={isLoading}
          dataSource={tableData}
          columns={columns}
          pagination={{
            current: page,
            pageSize,
            total: data?.total ?? 0,
            showSizeChanger: true,
            onChange: (p, ps) => {
              setPage(p);
              setPageSize(ps);
            },
          }}
          scroll={{ x: 860 }}
        />
      </Card>

      <CreateUserModal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        onSuccess={() => refetch()}
        roleLevel={roleLevel}
        restrictDeptId={restrictDeptId}
      />
    </>
  );
}
