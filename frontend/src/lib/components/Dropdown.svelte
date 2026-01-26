<script lang="ts">
  interface Option {
    label: string;
    value: string; // Ensure values are strings for proper binding
  }

  const defaultOption: Option = { label: 'Select an option', value: '' };
  export let options: Option[];
  export let selectedOption: Option = defaultOption;

  function handleSelectChange(event: Event) {
    const target = event.target as HTMLSelectElement;
    selectedOption = options.find((option) => option.value === target.value) ?? defaultOption;
  }
</script>

<select on:change={handleSelectChange} bind:value={selectedOption.value}>
  <option value="" disabled={!selectedOption} selected={!selectedOption}> Select an option </option>
  {#each options as option}
    <option value={option.value} selected={selectedOption?.value === option.value}>
      {option.label}
    </option>
  {/each}
</select>
