import React from "react";
import { Form } from "react-bootstrap";
import {
  Criterion,
  CriterionValue,
} from "../../../models/list-filter/criteria/criterion";

interface IOptionsFilterProps {
  criterion: Criterion<CriterionValue>;
  onValueChanged: (value: CriterionValue) => void;
}

export const OptionsFilter: React.FC<IOptionsFilterProps> = ({
  criterion,
  onValueChanged,
}) => {
  function onChanged(event: React.ChangeEvent<HTMLSelectElement>) {
    onValueChanged(event.target.value);
  }

  const options = criterion.criterionOption.options ?? [];

  if (
    options &&
    (criterion.value === undefined ||
      criterion.value === "" ||
      typeof criterion.value === "number")
  ) {
    onValueChanged(options[0].toString());
  }

  return (
    <Form.Group>
      <Form.Control
        as="select"
        onChange={onChanged}
        value={criterion.value.toString()}
        className="btn-secondary"
      >
        {options.map((c) => (
          <option key={c.toString()} value={c.toString()}>
            {c}
          </option>
        ))}
      </Form.Control>
    </Form.Group>
  );
};
