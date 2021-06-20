import React, { useEffect } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import * as GQL from "src/core/generated-graphql";
import * as yup from "yup";
import { DetailsEditNavbar, TagSelect } from "src/components/Shared";
import { Form, Col, Row } from "react-bootstrap";
import { FormUtils, ImageUtils } from "src/utils";
import { useFormik } from "formik";
import { Prompt, useHistory } from "react-router-dom";
import Mousetrap from "mousetrap";
import { StringListInput } from "src/components/Shared/StringListInput";

interface ITagEditPanel {
  tag?: Partial<GQL.TagDataFragment>;
  // returns id
  onSubmit: (
    tag: Partial<GQL.TagCreateInput | GQL.TagUpdateInput>
  ) => Promise<string | undefined>;
  onCancel: () => void;
  onDelete: () => void;
  setImage: (image?: string | null) => void;
}

export const TagEditPanel: React.FC<ITagEditPanel> = ({
  tag,
  onSubmit,
  onCancel,
  onDelete,
  setImage,
}) => {
  const intl = useIntl();
  const history = useHistory();

  const isNew = tag === undefined;

  const labelXS = 3;
  const labelXL = 3;
  const fieldXS = 9;
  const fieldXL = 9;

  const schema = yup.object({
    name: yup.string().required(),
    aliases: yup
      .array(yup.string().required())
      .optional()
      .test({
        name: "unique",
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        test: (value: any) => {
          return (value ?? []).length === new Set(value).size;
        },
        message: "aliases must be unique",
      }),
    parent_ids: yup.array(yup.string().required()).optional().nullable(),
    child_ids: yup.array(yup.string().required()).optional().nullable(),
  });

  const initialValues = {
    name: tag?.name,
    aliases: tag?.aliases,
    parent_ids: (tag?.parents ?? []).map((t) => t.id),
    child_ids: (tag?.children ?? []).map((t) => t.id),
  };

  type InputValues = typeof initialValues;

  const formik = useFormik({
    initialValues,
    validationSchema: schema,
    enableReinitialize: true,
    onSubmit: doSubmit,
  });

  async function doSubmit(values: InputValues) {
    const id = await onSubmit(getTagInput(values));
    if (id) {
      formik.resetForm({ values });
      history.push(`/tags/${id}`);
    }
  }

  // set up hotkeys
  useEffect(() => {
    Mousetrap.bind("s s", () => formik.handleSubmit());

    return () => {
      Mousetrap.unbind("s s");
    };
  });

  function getTagInput(values: InputValues) {
    const input: Partial<GQL.TagCreateInput | GQL.TagUpdateInput> = {
      ...values,
    };

    if (tag && tag.id) {
      (input as GQL.TagUpdateInput).id = tag.id;
    }
    return input;
  }

  function onImageChange(event: React.FormEvent<HTMLInputElement>) {
    ImageUtils.onImageChange(event, setImage);
  }

  const isEditing = true;

  // TODO: CSS class
  return (
    <div>
      {isNew && (
        <h2>
          <FormattedMessage
            id="actions.add_entity"
            values={{ entityType: intl.formatMessage({ id: "tag" }) }}
          />
        </h2>
      )}

      <Prompt
        when={formik.dirty}
        message={(location) => {
          if (!isNew && location.pathname.startsWith(`/tags/${tag?.id}`)) {
            return true;
          }
          return "Unsaved changes. Are you sure you want to leave?";
        }}
      />

      <Form noValidate onSubmit={formik.handleSubmit} id="tag-edit">
        <Form.Group controlId="name" as={Row}>
          <Form.Label column xs={labelXS} xl={labelXL}>
            <FormattedMessage id="name" />
          </Form.Label>
          <Col xs={fieldXS} xl={fieldXL}>
            <Form.Control
              className="text-input"
              placeholder="Name"
              {...formik.getFieldProps("name")}
              isInvalid={!!formik.errors.name}
            />
            <Form.Control.Feedback type="invalid">
              {formik.errors.name}
            </Form.Control.Feedback>
          </Col>
        </Form.Group>

        <Form.Group controlId="aliases" as={Row}>
          <Form.Label column xs={labelXS} xl={labelXL}>
            <FormattedMessage id="aliases" />
          </Form.Label>
          <Col xs={fieldXS} xl={fieldXL}>
            <StringListInput
              value={formik.values.aliases ?? []}
              setValue={(value) => formik.setFieldValue("aliases", value)}
              errors={formik.errors.aliases}
            />
          </Col>
        </Form.Group>

        <Form.Group controlId="parent_tags" as={Row}>
          {FormUtils.renderLabel({
            title: intl.formatMessage({ id: "parent_tags" }),
            labelProps: {
              column: true,
              sm: 3,
              xl: 12,
            },
          })}
          <Col sm={9} xl={12}>
            <TagSelect
              isMulti
              onSelect={(items) =>
                formik.setFieldValue(
                  "parent_ids",
                  items.map((item) => item.id)
                )
              }
              ids={formik.values.parent_ids}
              excludeIds={(tag?.id ? [tag.id] : []).concat(
                ...formik.values.child_ids
              )}
              creatable={false}
            />
          </Col>
        </Form.Group>

        <Form.Group controlId="child_tags" as={Row}>
          {FormUtils.renderLabel({
            title: intl.formatMessage({ id: "child_tags" }),
            labelProps: {
              column: true,
              sm: 3,
              xl: 12,
            },
          })}
          <Col sm={9} xl={12}>
            <TagSelect
              isMulti
              onSelect={(items) =>
                formik.setFieldValue(
                  "child_ids",
                  items.map((item) => item.id)
                )
              }
              ids={formik.values.child_ids}
              excludeIds={(tag?.id ? [tag.id] : []).concat(
                ...formik.values.parent_ids
              )}
              creatable={false}
            />
          </Col>
        </Form.Group>
      </Form>

      <DetailsEditNavbar
        objectName={tag?.name ?? "tag"}
        isNew={isNew}
        isEditing={isEditing}
        onToggleEdit={onCancel}
        onSave={() => formik.handleSubmit()}
        onImageChange={onImageChange}
        onImageChangeURL={setImage}
        onClearImage={() => {
          setImage(null);
        }}
        onDelete={onDelete}
      />
    </div>
  );
};
