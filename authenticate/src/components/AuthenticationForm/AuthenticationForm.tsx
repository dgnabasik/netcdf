// AuthenticationForm.tsx   https://mailboxlayer.com/documentation
import React, { useState, forwardRef } from "react";
import { useForm } from "react-hook-form";
import "./authenticationForm.css";
import validator from 'validator';

// https://email-checker.net/check  https://www.verifyemailaddress.org/
// https://www.digitalocean.com/community/tutorials/how-to-set-up-ssh-keys-on-ubuntu-20-04
// TODO: Become a member of list of Checkboxes for available Group.ID+names.
// TODO: output rdf code; button to upload.

export interface IAuthenticationProps {
  options?: { mbox: string; maker: string; password: string; value: string | number }[];
  mbox: string;
  maker: string;
  password: string;
}; 

function AuthenticationForm() {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm();
  
  const validateEmail = async (email: string) => {
    const apiKey:string = "8edb0e203e0c2d1a9e87746f64de9a18";
    const apiAccount:string = " david.gnabasik@emse.fr";
    const apiUrl: string = "http://apilayer.net/api/check?access_key=" + apiKey + "&email=" + apiAccount + "&smtp=1&format=1";
    const response = await fetch(apiUrl);
    const mboxJson = await response.json(); //extract JSON from the http response
    const result = !mboxJson.hasOwnProperty("success");
    console.log(result);//<<<
    return result;
  }

  const onSubmit = (data: IAuthenticationProps) => {
    console.log(data);
  };

  const [errorMessage, setErrorMessage] = useState('')

  const validate = (value) => {  
    console.log(value);
    if (validator.isStrongPassword(value, {
        minLength: 8, minLowercase: 1,
        minUppercase: 1, minNumbers: 1, minSymbols: 1
    })) {
        setErrorMessage('Password is strong!')
    } else {
        setErrorMessage('Password is not strong enough...')
    }
  }

//  const Select = forwardRef<HTMLSelectElement, { label: string } , password & ReturnType<UseFormRegister<IFormValues>>>(({ onChange, onBlur, name, label }, ref) => (
  const Select = forwardRef<HTMLSelectElement, IAuthenticationProps> (({ options, password }, ref) => {  
    return (
      <>
      <label>{label}</label>
      <select password={password} ref={ref} onChange={onChange} onBlur={onBlur}>
      {options?.map((op) => (<option value={op.value}>{op.password}</option>))}
      </select>
      </>
    );
  });

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="hook">
      <label className="hook__text">emse.fr Email</label>
      <input
        type="email"
        className="hook__input"
        {...register("mbox", { required: true, pattern: /^\S+@\S+$/i })}
      />
      {errors.mbox && (
        <p className="hook__error">emse.fr email is required and must be valid</p>
      )}

      <label className="hook__text">Unique rdf:resource Identifier such as FirstName+LastName (no dot)</label>
      <input
        type="text"
        className="hook__input"
        {...register("maker", { required: true })}
      />
      {errors.maker && (
        <p className="hook__error">Identifier is required and must be valid</p>
      )}

      <label className="hook__text">Password</label>
      <input
        type="password"
        className="hook__input"
        onChange={(e) => validate(e.target.value)}
        {...register("password", { required: true })}        
      />
      {errors.password && <p className="hook__error">Password is required</p>}

      <label className="hook__text">You will be granted Administrative access if you enter your complete (3 pieces) public key that you generated on your computer using ssh-keygen.</label>
      <label className="hook__text">Open or cat that file, such as ~/.ssh/[mynewkey].pub  This public key will be added to the authentication server.</label>
      <input
        type="text"
        className="hook__input"
        {...register("sha1", { required: false })}
      />

      <button className="hook__button" type="submit">
        Submit
      </button>
    </form>
  );
}

export default AuthenticationForm;
/*
<rdf:RDF
      xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
      xmlns:rdfs="http://www.w3.org/2000/01/rdf-schema#"
      xmlns:foaf="http://xmlns.com/foaf/0.1/"
      xmlns:dc="http://purl.org/dc/elements/1.1/"
      xmlns:bibo="http://purl.org/ontology/bibo/"
      xmlns:dcterms="http://purl.org/dc/terms/"
      xmlns:ical="http://www.w3.org/2002/12/cal/ical#"
      xmlns:geo="http://www.w3.org/2003/01/geo/wgs84_pos#" >

  <foaf:PersonalProfileDocument rdf:about="">
    <foaf:maker rdf:resource="#DavidGnabasik"/>
    <foaf:primaryTopic rdf:resource="#DavidGnabasik"/>
  </foaf:PersonalProfileDocument>

 <foaf:Person>
   <foaf:name>David Gnabasik</foaf:name>
   <foaf:Organization>École des Mines de Saint-Étienne</foaf:Organization>
   <foaf:mbox>David.Gnabasik@emse.fr</foaf:mbox>
   <foaf:sha1>ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQDb6/X7MkDXYNZma3jfgmbTZm/GsiGZvzi7fI9Tln6cPoxqdNqWwxztJDZp8yLv1tGuCKnE+6MIbZxXvqoPMCYasJjDeGARLKHlORKNCLcYf0AXDdegrsbp71igr+p8jhPXO5wZAfw5h+746or6fuzynqL/m2bgZpJfiQ4mZpSftf4+fuC/S/0TZNo7VBnzDa18W05eWHPLmyBvwCg/SMe8HVOqfqDEz3CSlNwebs/rU6ca2YPkVit/8OWR5STJBoVjsBAUiQI2WYy+TR6BxQ0FhxvH1ZlDfaRuMKotnn5Wqt3HFHKrLW3F+GrPpRDkjQlHpt2vGcJbvHgmBw79QTgoLgiaCbiss0wCRB5DG3Dt7uGKskyFZ9tQg4gVd5wvO6Xop65TAFxDr5Vjphbf4eQwtXcXBrhKJQ+QGinAEthl30u0k0SY268taj9ZFdI7JEOMMevA6hiuuFkPlYj9/dg+6DLUeBLyuwhvwKo3TjJFFl6htgEQAPm2NcKJr7LMu/0= david@david-NUC10i5FNH</foaf:sha1>

   <!-- knows -->
   <foaf:knows rdf:resource="#IotDataStreamingServer"/>
   <foaf:knows rdf:resource="#ChatGuillaume"/>

 </foaf:Person>

  <!-- groups -->
  <foaf:Group rdf:ID="IotDataStreamingServer">
    <foaf:name>IoT Data Streaming Server</foaf:name>
    <foaf:member rdf:resource="#DavidGnabasik"/>
  </foaf:Group>

  <foaf:Group rdf:ID="ChatGuillaume">
    <foaf:name>Chat Guillaume</foaf:name>
    <foaf:member rdf:resource="#DavidGnabasik"/>
  </foaf:Group>

</rdf:RDF>
*/