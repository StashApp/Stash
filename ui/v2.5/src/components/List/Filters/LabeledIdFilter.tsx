import React from "react";
import { Form } from "react-bootstrap";
import { FilterSelect, ValidTypes } from "../../Shared";
import { Criterion } from "../../../models/list-filter/criteria/criterion";
import { ILabeledId } from "../../../models/list-filter/types";

interface ILabeledIdFilterProps {
  criterion: Criterion<ILabeledId[]>;
  onValueChanged: (value: ILabeledId[]) => void;
}

export const LabeledIdFilter: React.FC<ILabeledIdFilterProps> = ({
  criterion,
  onValueChanged,
}) => {
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

  function onSelectionChanged(items: ValidTypes[]) {
    onValueChanged(
      items.map((i) => ({
        id: i.id,
        label: i.name!,
      }))
    );
  }

  return (
    <Form.Group>
      <FilterSelect
        type={criterion.criterionOption.type}
        isMulti
        onSelect={onSelectionChanged}
        ids={criterion.value.map((labeled) => labeled.id)}
      />
    </Form.Group>
  );
};
