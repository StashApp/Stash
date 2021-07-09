import _ from "lodash";
import React, { useEffect, useRef, useState } from "react";
import { Button, Form, Modal } from "react-bootstrap";
import { CriterionModifier } from "src/core/generated-graphql";
import {
  DurationCriterion,
  CriterionValue,
  Criterion,
  IHierarchicalLabeledIdCriterion,
  NumberCriterion,
  ILabeledIdCriterion,
} from "src/models/list-filter/criteria/criterion";
import {
  NoneCriterion,
  NoneCriterionOption,
} from "src/models/list-filter/criteria/none";
import { makeCriteria } from "src/models/list-filter/criteria/factory";
import { ListFilterOptions } from "src/models/list-filter/filter-options";
import { FormattedMessage, useIntl } from "react-intl";
import {
  criterionIsHierarchicalLabelValue,
  criterionIsNumberValue,
  CriterionType,
} from "src/models/list-filter/types";
import { DurationFilter } from "./Filters/DurationFilter";
import { NumberFilter } from "./Filters/NumberFilter";
import { LabeledIdFilter } from "./Filters/LabeledIdFilter";
import { HierarchicalLabelValueFilter } from "./Filters/HierarchicalLabelValueFilter";
import { OptionsFilter } from "./Filters/OptionsFilter";
import { InputFilter } from "./Filters/InputFilter";

interface IAddFilterProps {
  onAddCriterion: (
    criterion: Criterion<CriterionValue>,
    oldId?: string
  ) => void;
  onCancel: () => void;
  filterOptions: ListFilterOptions;
  editingCriterion?: Criterion<CriterionValue>;
}

export const AddFilterDialog: React.FC<IAddFilterProps> = ({
  onAddCriterion,
  onCancel,
  filterOptions,
  editingCriterion,
}) => {
  const defaultValue = useRef<string | number | undefined>();

  const [criterion, setCriterion] = useState<Criterion<CriterionValue>>(
    new NoneCriterion()
  );
  const { options, modifierOptions } = criterion.criterionOption;

  const valueStage = useRef<CriterionValue>(criterion.value);

  const intl = useIntl();

  // Configure if we are editing an existing criterion
  useEffect(() => {
    if (!editingCriterion) {
      setCriterion(makeCriteria());
    } else {
      setCriterion(editingCriterion);
    }
  }, [editingCriterion]);

  useEffect(() => {
    valueStage.current = criterion.value;
  }, [criterion]);

  function onChangedCriteriaType(event: React.ChangeEvent<HTMLSelectElement>) {
    const newCriterionType = event.target.value as CriterionType;
    const newCriterion = makeCriteria(newCriterionType);
    setCriterion(newCriterion);
  }

  function onChangedModifierSelect(
    event: React.ChangeEvent<HTMLSelectElement>
  ) {
    const newCriterion = _.cloneDeep(criterion);
    newCriterion.modifier = event.target.value as CriterionModifier;
    setCriterion(newCriterion);
  }

  function onValueChanged(value: CriterionValue) {
    const newCriterion = _.cloneDeep(criterion);
    newCriterion.value = value;
    setCriterion(newCriterion);
  }

  function onAddFilter() {
    const oldId = editingCriterion ? editingCriterion.getId() : undefined;
    onAddCriterion(criterion, oldId);
  }

  const maybeRenderFilterPopoverContents = () => {
    if (criterion.criterionOption.type === "none") {
      return;
    }

    function renderModifier() {
      if (modifierOptions.length === 0) {
        return;
      }
      return (
        <Form.Control
          as="select"
          onChange={onChangedModifierSelect}
          value={criterion.modifier}
          className="btn-secondary"
        >
          {modifierOptions.map((c) => (
            <option key={c.value} value={c.value}>
              {c.label ? intl.formatMessage({ id: c.label }) : ""}
            </option>
          ))}
        </Form.Control>
      );
    }

    function renderSelect() {
      // Hide the value select if the modifier is "IsNull" or "NotNull"
      if (
        criterion.modifier === CriterionModifier.IsNull ||
        criterion.modifier === CriterionModifier.NotNull
      ) {
        return;
      }

      if (criterion instanceof ILabeledIdCriterion) {
        return (
          <LabeledIdFilter
            criterion={criterion}
            onValueChanged={onValueChanged}
          />
        );
      }
      if (criterion instanceof IHierarchicalLabeledIdCriterion) {
        return (
          <HierarchicalLabelValueFilter
            criterion={criterion}
            onValueChanged={onValueChanged}
          />
        );
      }
      if (
        options &&
        !criterionIsHierarchicalLabelValue(criterion.value) &&
        !criterionIsNumberValue(criterion.value) &&
        !Array.isArray(criterion.value)
      ) {
        defaultValue.current = criterion.value;
        return (
          <OptionsFilter
            criterion={criterion}
            onValueChanged={onValueChanged}
          />
        );
      }
      if (criterion instanceof DurationCriterion) {
        return (
          <DurationFilter
            criterion={criterion}
            onValueChanged={onValueChanged}
          />
        );
      }
      if (criterion instanceof NumberCriterion) {
        return (
          <NumberFilter criterion={criterion} onValueChanged={onValueChanged} />
        );
      }
      return (
        <InputFilter criterion={criterion} onValueChanged={onValueChanged} />
      );
    }
    return (
      <>
        <Form.Group>{renderModifier()}</Form.Group>
        {renderSelect()}
      </>
    );
  };

  function maybeRenderFilterCriterion() {
    if (!editingCriterion) {
      return;
    }

    return (
      <Form.Group>
        <strong>
          {intl.formatMessage({
            id: editingCriterion.criterionOption.messageID,
          })}
        </strong>
      </Form.Group>
    );
  }

  function maybeRenderFilterSelect() {
    if (editingCriterion) {
      return;
    }

    const thisOptions = [NoneCriterionOption]
      .concat(filterOptions.criterionOptions)
      .map((c) => {
        return {
          value: c.type,
          text: intl.formatMessage({ id: c.messageID }),
        };
      })
      .sort((a, b) => {
        if (a.value === "none") return -1;
        if (b.value === "none") return 1;
        return a.text.localeCompare(b.text);
      });

    return (
      <Form.Group controlId="filter">
        <Form.Label>
          <FormattedMessage id="search_filter.name" />
        </Form.Label>
        <Form.Control
          as="select"
          onChange={onChangedCriteriaType}
          value={criterion.criterionOption.type}
          className="btn-secondary"
        >
          {thisOptions.map((c) => (
            <option key={c.value} value={c.value} disabled={c.value === "none"}>
              {c.text}
            </option>
          ))}
        </Form.Control>
      </Form.Group>
    );
  }

  const title = !editingCriterion
    ? intl.formatMessage({ id: "search_filter.add_filter" })
    : intl.formatMessage({ id: "search_filter.update_filter" });
  return (
    <>
      <Modal show onHide={() => onCancel()}>
        <Modal.Header>{title}</Modal.Header>
        <Modal.Body>
          <div className="dialog-content">
            {maybeRenderFilterSelect()}
            {maybeRenderFilterCriterion()}
            {maybeRenderFilterPopoverContents()}
          </div>
        </Modal.Body>
        <Modal.Footer>
          <Button
            onClick={onAddFilter}
            disabled={criterion.criterionOption.type === "none"}
          >
            {title}
          </Button>
        </Modal.Footer>
      </Modal>
    </>
  );
};
