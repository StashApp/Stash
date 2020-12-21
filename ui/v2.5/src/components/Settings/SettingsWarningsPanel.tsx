import React, { useState } from "react";
import { uniq } from "lodash";
import { Form, Row, Table } from "react-bootstrap";
import { Link } from "react-router-dom";

import * as GQL from "src/core/generated-graphql";
import { LoadingIndicator, TruncatedText } from "src/components/Shared";

export const SettingsWarningsPanel: React.FC = () => {
  const { data, loading } = GQL.useGetSceneErrorsQuery();
  const [filter, setFilter] = useState("");
  const [query, setQuery] = useState("");

  const scenes = data?.getSceneErrors ?? [];
  const types = uniq(scenes.map((s) => s.error_type));
  const filteredScenes = scenes.filter(
    (s) =>
      (filter === "" || filter === s.error_type) &&
      (query === "" ||
        s.scene?.title?.includes(query) ||
        s.scene?.path.includes(query) ||
        s.details.includes(query))
  );

  const handleFilterChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    setFilter(e.currentTarget.value);
  };

  const handleQueryChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setQuery(e.currentTarget.value);
  };

  if (loading) return <LoadingIndicator />;

  return (
    <div className="container">
      <Row>
        <h4>Warnings</h4>
      </Row>
      <Form>
        <Form.Group className="row align-items-center mt-4">
          <Form.Control
            onChange={handleQueryChange}
            className="col-4 input-control"
            placeholder="Filter..."
          />
          <Form.Label className="ml-auto mr-4">Show warning types:</Form.Label>
          <Form.Control
            as="select"
            custom
            onChange={handleFilterChange}
            className="col-2 input-control"
          >
            <option value="">All</option>
            {types.map((t) => (
              <option value={t}>{t}</option>
            ))}
          </Form.Control>
        </Form.Group>
      </Form>
      <Row>
        <Table striped className="warnings-table">
          <thead>
            <tr>
              <th className="warnings-table-scene">Scene</th>
              <th className="warnings-table-type">Warning Type</th>
              <th className="warning-table-details">Details</th>
            </tr>
          </thead>
          <tbody>
            {filteredScenes.map((e) => (
              <tr>
                <td>
                  {e.scene ? (
                    <Link to={`/scenes/${e.scene.id}`}>{e.scene.path}</Link>
                  ) : (
                    "None"
                  )}
                </td>
                <td>{e.error_type}</td>
                <td className="warnings-table-details">
                  {e.related_scene ? (
                    <Link to={`/scenes/${e.related_scene.id}`}>
                      {e.related_scene.path}
                    </Link>
                  ) : (
                    <TruncatedText text={e.details} lineCount={3} />
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </Table>
      </Row>
      {filteredScenes.length === 0 && (
        <h4 className="text-center mt-4">No warnings found.</h4>
      )}
    </div>
  );
};
