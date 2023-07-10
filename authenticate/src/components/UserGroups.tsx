// UserGroups.tsx  
import { useState } from "react";
import Checkbox from "./Checkbox";

function UserGroups({UserGroupProps}) {
  const [isChecked, setIsChecked] = useState(false);
  const userGroupProps: Map<string, string> = UserGroupProps;

  const handleChange = (event) => {
    setIsChecked(event.target.checked);
  };

  function dynamicCheckboxes() {
    Array.from(userGroupProps.entries()).map((entry) => {
      const [key, value] = entry;
      return (<Checkbox cboxKey={key} cboxValue={value} />);
    }
  )};

  return (
    <div>
      && dynamicCheckboxes()
    </div>
  );
}
//       {isChecked && <div>Blue is selected!</div>}

export default UserGroups;
