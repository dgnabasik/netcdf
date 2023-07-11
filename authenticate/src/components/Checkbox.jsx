// Checkbox.jsx
import { useState } from "react";
import "./css/checkbox.css";

function Checkbox(cboxKey, cboxValue) {
  const [isChecked, setIsChecked] = useState(false);
console.log(cboxKey +":"+cboxValue);//<<<
  const handleChange = (event) => {
    setIsChecked(event.target.checked);
  };

  return (
    <form>
      <label htmlFor="color" className="checkbox__text">
        <input
          className="checkbox__input"
          type="checkbox"
          name={cboxKey}
          checked={isChecked}
          onChange={handleChange}
        />
        {cboxValue}
      </label>
    </form>
  );
}
//       {isChecked && <div>Blue is selected!</div>}

export default Checkbox;
