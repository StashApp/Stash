import React from "react";
import { Form } from "react-bootstrap";
import { defineMessages, useIntl } from "react-intl";
import { FilterSelect, ValidTypes } from "../../Shared";
import { Criterion } from "../../../models/list-filter/criteria/criterion";
import { IHierarchicalLabelValue } from "../../../models/list-filter/types";

interface IHierarchicalLabelValueFilterProps {
  criterion: Criterion<IHierarchicalLabelValue>;
  onValueChanged: (value: IHierarchicalLabelValue) => void;
}

export const HierarchicalLabelValueFilter: React.FC<IHierarchicalLabelValueFilterProps> = ({
  criterion,
  onValueChanged,
}) => {
  const intl = useIntl();

  if (
    criterion.criterionOption.type !== "performers" &&
    criterion.criterionOption.type !== "studios" &&
    criterion.criterionOption.type !== "parent_studios" &&
    criterion.criterionOption.type !== "tags" &&
    criterion.criterionOption.type !== "sceneTags" &&
    criterion.criterionOption.type !== "performerTags" &&
    criterion.criterionOption.type !== "movies"
  )
    return null;

  const messages = defineMessages({
    studio_depth: {
      id: "studio_depth",
      defaultMessage: "Levels (empty for all)",
    },
  });

  function onSelectionChanged(items: ValidTypes[]) {
    const { value } = criterion;
    value.items = items.map((i) => ({
      id: i.id,
      label: i.name!,
    }));
    onValueChanged(value);
  }

  function onDepthChanged(depth: number) {
    const { value } = criterion;
    value.depth = depth;
    onValueChanged(value);
  }

  return (
    <>
      <Form.Group>
        <FilterSelect
          type={criterion.criterionOption.type}
          isMulti
          onSelect={onSelectionChanged}
          ids={criterion.value.items.map((labeled) => labeled.id)}
        />
      </Form.Group>

      <Form.Group>
        <Form.Check
          checked={criterion.value.depth !== 0}
          label={intl.formatMessage({ id: "include_child_studios" })}
          onChange={() => onDepthChanged(criterion.value.depth !== 0 ? 0 : -1)}
        />
      </Form.Group>

      {criterion.value.depth !== 0 && (
        <Form.Group>
          <Form.Control
            className="btn-secondary"
            type="number"
            placeholder={intl.formatMessage(messages.studio_depth)}
            onChange={(e) =>
              onDepthChanged(e.target.value ? parseInt(e.target.value, 10) : -1)
            }
            defaultValue={
              criterion.value && criterion.value.depth !== -1
                ? criterion.value.depth
                : ""
            }
            min="1"
          />
        </Form.Group>
      )}
    </>
  );
};
