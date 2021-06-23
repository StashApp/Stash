import React from "react";
import { Form } from "react-bootstrap";

import { Modal } from "src/components/Shared";

interface IProps {
  show: boolean;
  hide: () => void;
  handleBatchUpdate: () => void;
  isIdle: boolean;
  sceneCount: number;
  setQueryAll: (queryAll: boolean) => void;
  setRefresh: (refresh: boolean) => void;
}

const BatchModal: React.FC<IProps> = ({
  show,
  hide,
  isIdle,
  handleBatchUpdate,
  sceneCount,
  setQueryAll,
  setRefresh,
}) => {
  return (
    <Modal
      show={show}
      icon="tags"
      header="Update Scenes"
      accept={{ text: "Update Scenes", onClick: handleBatchUpdate }}
      cancel={{
        text: "Cancel",
        variant: "danger",
        onClick: hide,
      }}
      disabled={!isIdle}
    >
      <Form.Group>
        <Form.Label>
          <h6>Scene selection</h6>
        </Form.Label>
        <Form.Check
          id="query-page"
          type="radio"
          name="scene-query"
          label="Current page"
          defaultChecked
          onChange={() => setQueryAll(false)}
        />
        <Form.Check
          id="query-all"
          type="radio"
          name="scene-query"
          label="All scenes in the database"
          defaultChecked={false}
          onChange={() => setQueryAll(true)}
        />
      </Form.Group>
      <Form.Group>
        <Form.Label>
          <h6>Tag Status</h6>
        </Form.Label>
        <Form.Check
          id="untagged-scenes"
          type="radio"
          name="scene-refresh"
          label="Untagged scenes"
          defaultChecked
          onChange={() => setRefresh(false)}
        />
        <Form.Text>
          Updating untagged scenes will try to match any scenes that lack a
          stashid and update the metadata.
        </Form.Text>
        <Form.Check
          id="tagged-scenes"
          type="radio"
          name="scene-refresh"
          label="Refresh tagged scenes"
          defaultChecked={false}
          onChange={() => setRefresh(true)}
        />
        <Form.Text>
          Refreshing will update the data of any tagged scenes from the
          stash-box instance.
        </Form.Text>
      </Form.Group>
      <b>{`${sceneCount} scenes will be processed`}</b>
    </Modal>
  );
};

export default BatchModal;
