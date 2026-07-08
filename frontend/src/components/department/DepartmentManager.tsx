import { useEffect, useMemo, useState } from 'react';
import { Button, Card, Col, Descriptions, Empty, Result, Row, Skeleton, Tree } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useGetDeptTreeQuery } from '@/store/api/userApi';
import CreateDeptModal from '@/components/department/CreateDeptModal';
import { findDeptNode, toAntdTreeData } from '@/utils/dept';

export default function DepartmentManager() {
  const { data: deptTree, isLoading, isError, refetch } = useGetDeptTreeQuery();
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [createOpen, setCreateOpen] = useState(false);

  const treeData = useMemo(() => (deptTree ? toAntdTreeData(deptTree) : []), [deptTree]);

  useEffect(() => {
    if (!deptTree?.length || selectedId !== null) return;
    setSelectedId(deptTree[0].id);
  }, [deptTree, selectedId]);

  const selectedNode = useMemo(() => {
    if (!deptTree || selectedId === null) return null;
    return findDeptNode(deptTree, selectedId);
  }, [deptTree, selectedId]);

  if (isError) {
    return (
      <Result
        status="error"
        title="组织树加载失败"
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
      <Row gutter={16}>
        <Col xs={24} lg={10}>
          <Card title="组织树">
            {isLoading ? (
              <Skeleton active paragraph={{ rows: 8 }} />
            ) : treeData.length === 0 ? (
              <Empty description="暂无组织数据" />
            ) : (
              <Tree
                treeData={treeData}
                selectedKeys={selectedId !== null ? [String(selectedId)] : []}
                onSelect={(keys) => {
                  const key = keys[0];
                  if (key) setSelectedId(Number(key));
                }}
                defaultExpandAll
                blockNode
              />
            )}
          </Card>
        </Col>
        <Col xs={24} lg={14}>
          <Card
            title="部门详情"
            extra={
              selectedNode ? (
                <Button icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
                  新增子部门
                </Button>
              ) : null
            }
          >
            {isLoading ? (
              <Skeleton active paragraph={{ rows: 4 }} />
            ) : selectedNode ? (
              <Descriptions column={1} bordered>
                <Descriptions.Item label="名称">{selectedNode.deptName}</Descriptions.Item>
                <Descriptions.Item label="代码">{selectedNode.deptCode}</Descriptions.Item>
                <Descriptions.Item label="路径">{selectedNode.path}</Descriptions.Item>
                <Descriptions.Item label="上级 ID">{selectedNode.parentId || '—'}</Descriptions.Item>
              </Descriptions>
            ) : (
              <Empty description="请在左侧选择部门" />
            )}
          </Card>
        </Col>
      </Row>

      <CreateDeptModal
        open={createOpen}
        parentId={selectedId}
        parentName={selectedNode?.deptName}
        onClose={() => setCreateOpen(false)}
        onSuccess={() => refetch()}
      />
    </>
  );
}
